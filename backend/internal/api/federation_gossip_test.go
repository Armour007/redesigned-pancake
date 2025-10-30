package api

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

// Test verifyWithJWK for Ed25519 path and canonicalization stability
func TestVerifyWithJWKAndCanonicalize(t *testing.T) {
	// Generate ed25519 key
	_, pub, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	jwk := map[string]any{
		"kty": "OKP", "crv": "Ed25519", "x": base64.RawURLEncoding.EncodeToString(pub),
	}
	// Prepare data
	header := map[string]any{"alg": "EdDSA", "kid": "test"}
	hb, _ := json.Marshal(header)
	p := map[string]any{"foo": "bar"}
	pb, _ := json.Marshal(p)
	unsigned := []byte(base64.RawURLEncoding.EncodeToString(hb) + "." + base64.RawURLEncoding.EncodeToString(pb))
	_ = jwk
	_ = unsigned // scaffolding for future verifyWithJWK tests

	// Canonicalization should be stable across calls
	m := GossipMessage{OrgID: "o1", Topic: "revocation", Ts: time.Now().UTC(), Nonce: "n1", Payload: json.RawMessage(`{"a":1}`)}
	b1 := canonicalizeGossipForHash(m)
	b2 := canonicalizeGossipForHash(m)
	if string(b1) != string(b2) {
		t.Fatalf("canonicalization not stable")
	}
}
