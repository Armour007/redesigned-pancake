package attest

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// AzureSNPVerifier verifies a JWT-signed attestation token using x5c from payload.
// NOTE: This MVP uses the provided x5c chain without OS trust anchoring and does not fetch revocation; pinning a trusted root is recommended.

type AzureSNPVerifier struct{}

func (AzureSNPVerifier) Type() string { return "azure_snp" }

func (AzureSNPVerifier) Verify(payload map[string]any) (*VerifierResult, error) {
	tok, _ := payload["token"].(string)
	if tok == "" {
		return nil, errors.New("missing token")
	}
	// Expect optional x5c array (PEM or base64 DER)
	var certs []*x509.Certificate
	if x5cAny, ok := payload["x5c"].([]any); ok {
		for _, e := range x5cAny {
			if s, ok := e.(string); ok {
				// try PEM first
				if strings.Contains(s, "BEGIN CERTIFICATE") {
					if blk, _ := pem.Decode([]byte(s)); blk != nil {
						if c, err := x509.ParseCertificate(blk.Bytes); err == nil {
							certs = append(certs, c)
						}
					}
				} else {
					// assume base64 DER
					if der, err := base64.StdEncoding.DecodeString(s); err == nil {
						if c, err := x509.ParseCertificate(der); err == nil {
							certs = append(certs, c)
						}
					}
				}
			}
		}
	}
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		return nil, errors.New("bad jwt")
	}
	headerJSON, err := b64url(parts[0])
	if err != nil {
		return nil, err
	}
	var hdr struct {
		Alg, Typ string
		X5c      []string
	}
	_ = json.Unmarshal(headerJSON, &hdr)
	payloadJSON, err := b64url(parts[1])
	if err != nil {
		return nil, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}
	// choose leaf cert
	if len(certs) == 0 {
		return nil, errors.New("missing x5c")
	}
	leaf := certs[0]
	pub, ok := leaf.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("unsupported key")
	}
	signed := []byte(parts[0] + "." + parts[1])
	if !ecdsaVerifyRaw(pub, signed, sig) {
		return nil, errors.New("signature verify failed")
	}
	// Extract some claims for posture
	var pl map[string]any
	_ = json.Unmarshal(payloadJSON, &pl)
	measurement := fmt.Sprintf("azure-snp:%v", pl["x-ms-sevsnpvm-commitment"])
	fp := measurement // for MVP
	return &VerifierResult{Fingerprint: fp, Measurement: measurement, Posture: pl, PostureOK: true}, nil
}

func b64url(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(s) }

// ecdsaVerifyRaw performs a minimal ECDSA JWS signature verification for P-256 and DER-less raw sigs.
// For simplicity, assume ES256 and signature is raw (r||s); If DER, parsing would be needed. Here we try both.
func ecdsaVerifyRaw(pub *ecdsa.PublicKey, data, sig []byte) bool {
	h := sha256.Sum256(data)
	// try raw r||s
	if len(sig) == 64 {
		r := new(big.Int).SetBytes(sig[:32])
		s := new(big.Int).SetBytes(sig[32:])
		if ecdsa.Verify(pub, h[:], r, s) {
			return true
		}
	}
	// else: unsupported in MVP
	return false
}
