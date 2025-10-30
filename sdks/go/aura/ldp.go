package aura

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"

	ld "github.com/piprate/json-gold/ld"
)

// DIDDocument minimal shape
type didVerificationMethod struct {
	ID           string          `json:"id"`
	Type         string          `json:"type"`
	Controller   string          `json:"controller"`
	PublicKeyJwk json.RawMessage `json:"publicKeyJwk"`
}
type didDoc struct {
	ID                 string                  `json:"id"`
	VerificationMethod []didVerificationMethod `json:"verificationMethod"`
}

func fetchDIDDoc(ctx context.Context, baseURL, did string) (didDoc, error) {
	u := strings.TrimRight(baseURL, "/") + "/resolve?did=" + did
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return didDoc{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return didDoc{}, fmt.Errorf("resolve status %d", resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	var out didDoc
	if err := json.Unmarshal(b, &out); err != nil {
		return didDoc{}, err
	}
	return out, nil
}

// NormalizeVCURDNA normalizes a JSON-LD VC (without proof) to N-Quads using URDNA2015
func NormalizeVCURDNA(vc map[string]any) (string, error) {
	// copy and ensure proof is not present
	cp := map[string]any{}
	for k, v := range vc {
		if k == "proof" {
			continue
		}
		cp[k] = v
	}
	proc := ld.NewJsonLdProcessor()
	opts := ld.NewJsonLdOptions("")
	opts.ProcessingMode = ld.JsonLd_1_1
	opts.Format = "application/n-quads"
	norm, err := proc.Normalize(cp, opts)
	if err != nil {
		return "", err
	}
	s, _ := norm.(string)
	return s, nil
}

// VerifyVCLDP verifies a JsonWebSignature2020 proof using detached JWS over URDNA2015 N-Quads
// If expectedOrgID or expectedOwner are non-empty, validates issuer and subject invariants.
func VerifyVCLDP(ctx context.Context, baseURL string, vc map[string]any, expectedOrgID, expectedOwner string) (bool, string) {
	// Extract proof
	pr, ok := vc["proof"].(map[string]any)
	if !ok {
		return false, "missing_proof"
	}
	jws, _ := pr["jws"].(string)
	vmID, _ := pr["verificationMethod"].(string)
	if jws == "" || vmID == "" {
		return false, "invalid_proof"
	}
	// Parse detached JWS: protected..signature
	parts := strings.Split(jws, ".")
	if len(parts) != 3 || parts[1] != "" {
		return false, "not_detached"
	}
	// Decode protected header
	phb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false, "bad_protected"
	}
	var ph map[string]any
	_ = json.Unmarshal(phb, &ph)
	alg, _ := ph["alg"].(string)
	b64, _ := ph["b64"].(bool)
	if !b64 {
		// ok; required for detached unencoded
	} else {
		return false, "b64_must_be_false"
	}
	// Normalize VC
	nquads, err := NormalizeVCURDNA(vc)
	if err != nil || nquads == "" {
		return false, "normalize_failed"
	}
	signingInput := parts[0] + "." + nquads
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return false, "bad_sig"
	}
	// Resolve DID and find verification method
	did := vmID
	if i := strings.Index(vmID, "#"); i > 0 {
		did = vmID[:i]
	}
	doc, err := fetchDIDDoc(ctx, baseURL, did)
	if err != nil {
		return false, "resolve_failed"
	}
	var vm *didVerificationMethod
	for i := range doc.VerificationMethod {
		if doc.VerificationMethod[i].ID == vmID {
			vm = &doc.VerificationMethod[i]
			break
		}
	}
	if vm == nil {
		return false, "vm_not_found"
	}
	var jwk map[string]any
	_ = json.Unmarshal(vm.PublicKeyJwk, &jwk)
	// Verify signature
	switch alg {
	case "EdDSA":
		if jwk["kty"] != "OKP" || jwk["crv"] != "Ed25519" {
			return false, "jwk_mismatch"
		}
		xb, err := base64.RawURLEncoding.DecodeString(jwk["x"].(string))
		if err != nil {
			return false, "bad_jwk"
		}
		if !ed25519.Verify(ed25519.PublicKey(xb), []byte(signingInput), sig) {
			return false, "bad_sig"
		}
	case "ES256":
		if jwk["kty"] != "EC" || jwk["crv"] != "P-256" {
			return false, "jwk_mismatch"
		}
		xb, err1 := base64.RawURLEncoding.DecodeString(jwk["x"].(string))
		yb, err2 := base64.RawURLEncoding.DecodeString(jwk["y"].(string))
		if err1 != nil || err2 != nil {
			return false, "bad_jwk"
		}
		pub := ecdsa.PublicKey{Curve: elliptic.P256(), X: new(big.Int).SetBytes(xb), Y: new(big.Int).SetBytes(yb)}
		// Convert JOSE r||s to DER
		if len(sig) != 64 {
			return false, "bad_sig"
		}
		rs := struct{ R, S *big.Int }{R: new(big.Int).SetBytes(sig[:32]), S: new(big.Int).SetBytes(sig[32:])}
		der, _ := asn1.Marshal(rs)
		h := sha256.Sum256([]byte(signingInput))
		if !ecdsa.VerifyASN1(&pub, h[:], der) {
			return false, "bad_sig"
		}
	default:
		return false, "unsupported_alg"
	}
	// Invariants
	if expectedOrgID != "" {
		iss, _ := vc["issuer"].(string)
		want := "did:aura:org:" + expectedOrgID
		if iss != want {
			return false, "issuer_mismatch"
		}
	}
	if expectedOwner != "" {
		cs, _ := vc["credentialSubject"].(map[string]any)
		if cs == nil {
			return false, "no_subject"
		}
		own, _ := cs["owner"].(string)
		if own != expectedOwner {
			return false, "owner_mismatch"
		}
	}
	return true, "ok"
}
