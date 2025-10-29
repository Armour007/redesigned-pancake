package crypto

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"
)

// SignAlgorithm identifiers for JWT headers
const (
	AlgEdDSA = "EdDSA"
	AlgES256 = "ES256"
)

// Signer signs JWT-like payloads (header.payload) and exposes public JWK material
type Signer interface {
	Algorithm() string
	KeyID() string
	PublicJWK(ctx context.Context) (map[string]any, error)
	// Sign returns the raw signature bytes appropriate for the algorithm (Ed25519: 64 bytes; ES256: JOSE r||s 64 bytes)
	Sign(ctx context.Context, unsigned []byte) ([]byte, error)
}

// TrustKeyRecord describes key material row for constructing signers
type TrustKeyRecord struct {
	Provider       string          // e.g., "local","aws","gcp","azure","vault"
	KeyRef         string          // e.g., ARN, resource name, KV key identifier, vault key name
	KeyVersion     string          // optional version (annotated in kid)
	Alg            string          // AlgEdDSA or AlgES256
	Kid            string          // preferred kid
	EncPriv        string          // ed25519 private key (base64url/base64) for local
	ProviderConfig json.RawMessage // provider-specific settings
	JWKPub         json.RawMessage // cached/public JWK
}

// NewSignerFromRecord creates a signer for the given trust key record
func NewSignerFromRecord(rec TrustKeyRecord) (Signer, error) {
	switch strings.ToLower(rec.Provider) {
	case "", "local":
		// local Ed25519 only
		if rec.Alg == "" || rec.Alg == AlgEdDSA {
			priv, kid, err := parseLocalEd25519(rec.EncPriv, rec.Kid)
			if err != nil {
				return nil, err
			}
			return &LocalEd25519Signer{priv: priv, kid: kid}, nil
		}
		return nil, errors.New("local provider supports only EdDSA")
	case "vault":
		return NewVaultSigner(rec)
	case "aws":
		return nil, errors.New("aws kms signer not implemented in this build")
	case "gcp":
		return nil, errors.New("gcp kms signer not implemented in this build")
	case "azure":
		return nil, errors.New("azure key vault signer not implemented in this build")
	default:
		return nil, errors.New("unknown provider")
	}
}

// ----- Local Ed25519 signer -----
type LocalEd25519Signer struct {
	priv ed25519.PrivateKey
	kid  string
}

func (s *LocalEd25519Signer) Algorithm() string { return AlgEdDSA }
func (s *LocalEd25519Signer) KeyID() string     { return s.kid }

func (s *LocalEd25519Signer) PublicJWK(ctx context.Context) (map[string]any, error) {
	pub := ed25519.PrivateKey(s.priv).Public().(ed25519.PublicKey)
	xb := base64.RawURLEncoding.EncodeToString(pub)
	return map[string]any{"kty": "OKP", "crv": "Ed25519", "alg": AlgEdDSA, "use": "sig", "kid": s.kid, "x": xb}, nil
}

func (s *LocalEd25519Signer) Sign(ctx context.Context, unsigned []byte) ([]byte, error) {
	sig := ed25519.Sign(s.priv, unsigned)
	return sig, nil
}

func parseLocalEd25519(encPriv, kid string) (ed25519.PrivateKey, string, error) {
	if encPriv == "" {
		return nil, "", errors.New("missing private key")
	}
	// Try base64url then base64
	var raw []byte
	var err error
	if raw, err = base64.RawURLEncoding.DecodeString(encPriv); err != nil {
		raw, err = base64.StdEncoding.DecodeString(encPriv)
		if err != nil {
			return nil, "", err
		}
	}
	if l := len(raw); l != ed25519.PrivateKeySize {
		return nil, "", errors.New("bad ed25519 private key length")
	}
	priv := ed25519.PrivateKey(raw)
	if strings.TrimSpace(kid) == "" {
		// derive a short kid from pub
		pub := priv.Public().(ed25519.PublicKey)
		sum := sha256.Sum256(pub)
		kid = base64.RawURLEncoding.EncodeToString(sum[:8])
	}
	return priv, kid, nil
}

// ----- Vault Transit signer (basic) -----
type VaultSigner struct {
	keyName string
	mount   string
	addr    string
	token   string
	alg     string
	kid     string
	jwk     map[string]any // optional cached JWK
	http    *http.Client
}

func NewVaultSigner(rec TrustKeyRecord) (Signer, error) {
	cfg := map[string]any{}
	if len(rec.ProviderConfig) > 0 {
		_ = json.Unmarshal(rec.ProviderConfig, &cfg)
	}
	mount, _ := cfg["mount"].(string)
	if mount == "" {
		mount = "transit"
	}
	addr := os.Getenv("VAULT_ADDR")
	if v, ok := cfg["addr"].(string); ok && v != "" {
		addr = v
	}
	tok := os.Getenv("VAULT_TOKEN")
	if v, ok := cfg["token"].(string); ok && v != "" {
		tok = v
	}
	if addr == "" || tok == "" {
		return nil, errors.New("vault addr/token not configured")
	}
	alg := rec.Alg
	if alg == "" {
		alg = AlgEdDSA
	}
	kid := rec.Kid
	if strings.TrimSpace(kid) == "" {
		kid = "vault:" + rec.KeyRef
		if rec.KeyVersion != "" {
			kid += "@" + rec.KeyVersion
		}
	}
	var jwk map[string]any
	if len(rec.JWKPub) > 0 {
		_ = json.Unmarshal(rec.JWKPub, &jwk)
	}
	return &VaultSigner{keyName: rec.KeyRef, mount: mount, addr: addr, token: tok, alg: alg, kid: kid, jwk: jwk, http: &http.Client{Timeout: 6 * time.Second}}, nil
}

func (s *VaultSigner) Algorithm() string { return s.alg }
func (s *VaultSigner) KeyID() string     { return s.kid }

func (s *VaultSigner) PublicJWK(ctx context.Context) (map[string]any, error) {
	if s.jwk != nil {
		return s.jwk, nil
	}
	// best-effort fetch
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(s.addr, "/")+"/v1/"+s.mount+"/keys/"+s.keyName, nil)
	req.Header.Set("X-Vault-Token", s.token)
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Data struct {
			Keys map[string]struct {
				PublicKey string `json:"public_key"`
			} `json:"keys"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	// Not all Vault responses include PEM; prefer configured jwk
	return nil, errors.New("vault public jwk not available; provide jwk_pub at creation")
}

func (s *VaultSigner) Sign(ctx context.Context, unsigned []byte) ([]byte, error) {
	// Vault expects base64 input; for ES256 we let Vault hash; for EdDSA it's pure EdDSA over input
	in := map[string]any{
		"input": base64.StdEncoding.EncodeToString(unsigned),
	}
	if s.alg == AlgES256 {
		in["hash_algorithm"] = "sha2-256"
		in["marshaling_algorithm"] = "jws"
		// Note: For ECDSA, Vault returns ASN.1 by default; marshaling_algorithm=jws requests JOSE format if supported
	}
	b, _ := json.Marshal(in)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(s.addr, "/")+"/v1/"+s.mount+"/sign/"+s.keyName, strings.NewReader(string(b)))
	req.Header.Set("X-Vault-Token", s.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Data struct {
			Signature string `json:"signature"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	sig := out.Data.Signature
	// Format: vault:vX:base64
	parts := strings.Split(sig, ":")
	rawB64 := parts[len(parts)-1]
	der, err := base64.StdEncoding.DecodeString(rawB64)
	if err != nil {
		return nil, err
	}
	if s.alg == AlgEdDSA {
		return der, nil // vault returns raw ed25519 sig bytes
	}
	// ES256: Vault may return ASN.1 DER; convert to JOSE r||s if needed
	// Try to parse ASN.1; if length is 64 already, assume JOSE
	if len(der) == 64 {
		return der, nil
	}
	r, sBig, ok := parseECDSADER(der)
	if !ok {
		return nil, errors.New("invalid ECDSA signature format")
	}
	// left-pad to 32 bytes
	rb := r.Bytes()
	sb := sBig.Bytes()
	sigOut := make([]byte, 64)
	copy(sigOut[32-len(rb):32], rb)
	copy(sigOut[64-len(sb):], sb)
	return sigOut, nil
}

// parseECDSADER parses a minimal ASN.1 ECDSA signature (r,s)
func parseECDSADER(b []byte) (*big.Int, *big.Int, bool) {
	// Very small DER parser for ECDSA signatures: 0x30 len 0x02 lenR R 0x02 lenS S
	if len(b) < 8 || b[0] != 0x30 {
		return nil, nil, false
	}
	i := 2 // skip SEQUENCE and length
	if i >= len(b) || b[i] != 0x02 {
		return nil, nil, false
	}
	i++
	if i >= len(b) {
		return nil, nil, false
	}
	lr := int(b[i])
	i++
	if i+lr > len(b) {
		return nil, nil, false
	}
	r := new(big.Int).SetBytes(b[i : i+lr])
	i += lr
	if i >= len(b) || b[i] != 0x02 {
		return nil, nil, false
	}
	i++
	if i >= len(b) {
		return nil, nil, false
	}
	ls := int(b[i])
	i++
	if i+ls > len(b) {
		return nil, nil, false
	}
	s := new(big.Int).SetBytes(b[i : i+ls])
	return r, s, true
}

// Helper to build EC P-256 public JWK from key
func ECP256PublicJWK(k *ecdsa.PublicKey, kid string) map[string]any {
	xb := k.X.Bytes()
	yb := k.Y.Bytes()
	// left pad to 32 bytes
	x := make([]byte, 32)
	y := make([]byte, 32)
	copy(x[32-len(xb):], xb)
	copy(y[32-len(yb):], yb)
	return map[string]any{"kty": "EC", "crv": "P-256", "alg": AlgES256, "use": "sig", "kid": kid, "x": base64.RawURLEncoding.EncodeToString(x), "y": base64.RawURLEncoding.EncodeToString(y)}
}
