package api

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func signHS256(t *testing.T, claims map[string]any, secret string) string {
	t.Helper()
	header := map[string]any{"alg": "HS256", "typ": "JWT"}
	hb, _ := json.Marshal(header)
	pb, _ := json.Marshal(claims)
	hb64 := base64.RawURLEncoding.EncodeToString(hb)
	pb64 := base64.RawURLEncoding.EncodeToString(pb)
	unsigned := hb64 + "." + pb64
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return unsigned + "." + sig
}

func TestValidateHS256JWT_BadFormat(t *testing.T) {
	os.Setenv("AURA_TRUST_TOKEN_SIGNING_KEY", "testsecret")
	valid, _, reason := validateHS256JWT("not-a-jwt")
	if valid || reason == "" {
		t.Fatalf("expected invalid format, got valid=%v reason=%q", valid, reason)
	}
}

func TestValidateHS256JWT_WrongSignature(t *testing.T) {
	os.Setenv("AURA_TRUST_TOKEN_SIGNING_KEY", "testsecret")
	claims := map[string]any{"exp": time.Now().Add(60 * time.Second).Unix(), "foo": "bar"}
	tok := signHS256(t, claims, "othersecret")
	valid, _, reason := validateHS256JWT(tok)
	if valid || reason != "invalid signature" {
		t.Fatalf("expected invalid signature, got valid=%v reason=%q", valid, reason)
	}
}

func TestValidateHS256JWT_Expired(t *testing.T) {
	os.Setenv("AURA_TRUST_TOKEN_SIGNING_KEY", "testsecret")
	claims := map[string]any{"exp": time.Now().Add(-10 * time.Second).Unix()}
	tok := signHS256(t, claims, "testsecret")
	valid, _, reason := validateHS256JWT(tok)
	if valid || reason != "token expired" {
		t.Fatalf("expected expired, got valid=%v reason=%q", valid, reason)
	}
}

func TestValidateHS256JWT_Success(t *testing.T) {
	os.Setenv("AURA_TRUST_TOKEN_SIGNING_KEY", "testsecret")
	claims := map[string]any{"exp": time.Now().Add(60 * time.Second).Unix(), "org_id": "o1", "context_hash": "abc"}
	tok := signHS256(t, claims, "testsecret")
	valid, out, reason := validateHS256JWT(tok)
	if !valid || reason != "" {
		t.Fatalf("expected valid, got valid=%v reason=%q", valid, reason)
	}
	if out["org_id"].(string) != "o1" {
		t.Fatalf("expected org_id 'o1', got %v", out["org_id"])
	}
}

func TestValidateJWT_EdDSA_Success(t *testing.T) {
	// deterministic private key: 32-byte seed of zeros
	seed := make([]byte, 32)
	pk := base64.RawURLEncoding.EncodeToString(seed)
	os.Setenv("AURA_TRUST_ED25519_PRIVATE_KEY", pk)
	// build EdDSA token
	header := map[string]any{"alg": "EdDSA", "typ": "JWT"}
	claims := map[string]any{"exp": time.Now().Add(60 * time.Second).Unix(), "org_id": "o2"}
	hb, _ := json.Marshal(header)
	pb, _ := json.Marshal(claims)
	b64 := func(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }
	unsigned := b64(hb) + "." + b64(pb)
	// Sign using the helper from jwks.go
	priv, _, _ := loadEd25519KeyFromEnv()
	sig := ed25519.Sign(priv, []byte(unsigned))
	tok := unsigned + "." + b64(sig)
	valid, out, reason := validateJWT(tok)
	if !valid || reason != "" {
		t.Fatalf("expected valid eddsa, got valid=%v reason=%q", valid, reason)
	}
	if out["org_id"].(string) != "o2" {
		t.Fatalf("expected org_id 'o2', got %v", out["org_id"])
	}
}
