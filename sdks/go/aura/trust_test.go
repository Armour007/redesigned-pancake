package aura

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type jwkOut struct {
	Kty string `json:"kty"`
	Crv string `json:"crv,omitempty"`
	Alg string `json:"alg,omitempty"`
	Use string `json:"use,omitempty"`
	Kid string `json:"kid,omitempty"`
	X   string `json:"x,omitempty"`
	Y   string `json:"y,omitempty"`
}

type jwksOut struct {
	Keys []jwkOut `json:"keys"`
}

func b64url(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func makeJWT(header, payload map[string]any, signer func(unsigned []byte) ([]byte, error)) (string, error) {
	hb, _ := json.Marshal(header)
	pb, _ := json.Marshal(payload)
	h := b64url(hb)
	p := b64url(pb)
	unsigned := []byte(h + "." + p)
	sig, err := signer(unsigned)
	if err != nil {
		return "", err
	}
	return h + "." + p + "." + b64url(sig), nil
}

func TestVerifyTrustTokenOffline_Ed25519_Valid(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	kid := "k-ed"
	jwks := jwksOut{Keys: []jwkOut{{Kty: "OKP", Crv: "Ed25519", Alg: "EdDSA", Kid: kid, X: b64url(pub)}}}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/.well-known/") {
			_ = json.NewEncoder(w).Encode(jwks)
			return
		}
		w.WriteHeader(404)
	}))
	defer ts.Close()

	exp := time.Now().Add(2 * time.Minute).Unix()
	tok, err := makeJWT(map[string]any{"alg": "EdDSA", "kid": kid}, map[string]any{"exp": exp, "jti": "j1"}, func(unsigned []byte) ([]byte, error) {
		sig := ed25519.Sign(priv, unsigned)
		return sig, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := VerifyTrustTokenOffline(context.Background(), ts.URL, tok, "", 0, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.Valid {
		t.Fatalf("expected valid, got: %+v", res)
	}
}

func TestVerifyTrustTokenOffline_ES256_Valid(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	kid := "k-es"
	x := b64url(priv.PublicKey.X.Bytes())
	y := b64url(priv.PublicKey.Y.Bytes())
	jwks := jwksOut{Keys: []jwkOut{{Kty: "EC", Crv: "P-256", Alg: "ES256", Kid: kid, X: x, Y: y}}}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/.well-known/") {
			_ = json.NewEncoder(w).Encode(jwks)
			return
		}
		w.WriteHeader(404)
	}))
	defer ts.Close()

	exp := time.Now().Add(2 * time.Minute).Unix()
	signer := func(unsigned []byte) ([]byte, error) {
		h := sha256.Sum256(unsigned)
		r, s, err := ecdsa.Sign(rand.Reader, priv, h[:])
		if err != nil {
			return nil, err
		}
		// raw r||s 32-byte each
		rb := r.Bytes()
		sb := s.Bytes()
		// pad to 32
		rpad := append(make([]byte, 32-len(rb)), rb...)
		spad := append(make([]byte, 32-len(sb)), sb...)
		return append(rpad, spad...), nil
	}
	tok, err := makeJWT(map[string]any{"alg": "ES256", "kid": kid}, map[string]any{"exp": exp, "jti": "j2"}, signer)
	if err != nil {
		t.Fatal(err)
	}
	res, err := VerifyTrustTokenOffline(context.Background(), ts.URL, tok, "", 0, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.Valid {
		t.Fatalf("expected valid, got: %+v", res)
	}
}

func TestFetchRevocations_ETagFlow(t *testing.T) {
	etag := "W/\"abc123\""
	items := []map[string]any{{"jti": "r1", "revoked_at": time.Now().UTC().Format(time.RFC3339)}}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/trust-tokens/revocations") {
			if inm := r.Header.Get("If-None-Match"); inm == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.Header().Set("ETag", etag)
			_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
			return
		}
		w.WriteHeader(404)
	}))
	defer ts.Close()

	// first fetch
	got, newEtag, notMod, err := FetchRevocations(context.Background(), ts.URL, "org1", "")
	if err != nil {
		t.Fatal(err)
	}
	if notMod {
		t.Fatalf("unexpected not modified")
	}
	if newEtag != etag {
		t.Fatalf("etag mismatch: %s", newEtag)
	}
	if len(got) != 1 || got[0].JTI != "r1" {
		t.Fatalf("unexpected items: %+v", got)
	}

	// second fetch with If-None-Match
	got2, newEtag2, notMod2, err := FetchRevocations(context.Background(), ts.URL, "org1", newEtag)
	if err != nil {
		t.Fatal(err)
	}
	if !notMod2 {
		t.Fatalf("expected not modified")
	}
	if newEtag2 != newEtag {
		t.Fatalf("etag should be same")
	}
	if got2 != nil && len(got2) != 0 {
		t.Fatalf("items should be empty on 304")
	}
}
