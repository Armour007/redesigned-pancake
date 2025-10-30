package api

import (
	"bytes"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"
	"time"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func TestIntrospect_HS256_MarkUsed_ReplayDetected(t *testing.T) {
	// Skip unless explicitly enabled to avoid env coupling? This is unit with sqlmock, so ok.
	secret := "test_secret"
	os.Setenv("AURA_TRUST_TOKEN_SIGNING_KEY", secret)
	t.Cleanup(func() { os.Unsetenv("AURA_TRUST_TOKEN_SIGNING_KEY") })

	// Build a minimal HS256 JWT
	unsigned := base64url(`{"alg":"HS256","typ":"JWT"}`) + "." + base64url(`{"org_id":"org1","jti":"jti1","exp":4102444800}`)
	sig := hmacB64(secret, unsigned)
	token := unsigned + "." + sig

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	database.DB = sqlx.NewDb(db, "sqlmock")

	// Expect insert ok then conflict (RowsAffected 1 then 0)
	ins := regexp.QuoteMeta(`INSERT INTO trust_token_jti(org_id, jti, exp_at) VALUES ($1,$2, to_timestamp($3)) ON CONFLICT DO NOTHING`)
	mock.ExpectExec(ins).WithArgs("org1", "jti1", int64(4102444800)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(ins).WithArgs("org1", "jti1", int64(4102444800)).WillReturnResult(sqlmock.NewResult(0, 0))

	gin.SetMode(gin.TestMode)
	// First call: valid
	w1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(w1)
	c1.Request = httptest.NewRequest(http.MethodPost, "/v2/tokens/introspect", nil)
	body1, _ := json.Marshal(map[string]any{"token": token, "mark_used": true})
	c1.Request.Body = ioNopCloser(bytes.NewReader(body1))
	c1.Request.Header.Set("Content-Type", "application/json")
	c1.Params = gin.Params{}
	IntrospectTrustToken(c1)
	if w1.Code != 200 {
		t.Fatalf("first call expected 200, got %d", w1.Code)
	}
	var r1 map[string]any
	_ = json.Unmarshal(w1.Body.Bytes(), &r1)
	if v, _ := r1["valid"].(bool); !v {
		t.Fatalf("first call should be valid, got: %s", w1.Body.String())
	}

	// Second call: replayed
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodPost, "/v2/tokens/introspect", nil)
	body2, _ := json.Marshal(map[string]any{"token": token, "mark_used": true})
	c2.Request.Body = ioNopCloser(bytes.NewReader(body2))
	c2.Request.Header.Set("Content-Type", "application/json")
	IntrospectTrustToken(c2)
	if w2.Code != 200 {
		t.Fatalf("second call expected 200, got %d", w2.Code)
	}
	var r2 map[string]any
	_ = json.Unmarshal(w2.Body.Bytes(), &r2)
	if v, _ := r2["valid"].(bool); v {
		t.Fatalf("second call should be invalid (replayed), got: %s", w2.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// helpers
func base64url(s string) string {
	return toB64URL([]byte(s))
}
func toB64URL(b []byte) string {
	enc := base64.RawURLEncoding.EncodeToString(b)
	return enc
}
func hmacB64(secret, unsigned string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	return toB64URL(mac.Sum(nil))
}

// tiny io.NopCloser substitute to avoid extra imports in test
type nopCloser struct{ r *bytes.Reader }

func (n nopCloser) Read(p []byte) (int, error) { return n.r.Read(p) }
func (n nopCloser) Close() error               { return nil }
func ioNopCloser(r *bytes.Reader) nopCloser    { return nopCloser{r} }

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
