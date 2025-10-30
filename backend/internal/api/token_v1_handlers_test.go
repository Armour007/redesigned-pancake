package api

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	database "github.com/Armour007/aura-backend/internal"
)

type issueResp struct {
	Token string `json:"token"`
	Alg   string `json:"alg"`
	Kid   string `json:"kid"`
	Exp   int64  `json:"exp"`
	JTI   string `json:"jti"`
}

type verifyResp struct {
	Valid  bool                   `json:"valid"`
	Reason string                 `json:"reason"`
	Claims map[string]interface{} `json:"claims"`
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/issue", IssueTrustTokenV1)
	r.POST("/verify", VerifyTrustTokenV1)
	return r
}

func TestIssueTrustTokenV1_HS256(t *testing.T) {
	// Mock DB: trust_keys SELECT returns no rows -> fallback path
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	database.DB = sqlx.NewDb(db, "sqlmock")
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT alg, COALESCE(kid,''), provider, key_ref, key_version, COALESCE(provider_config,'{}'::jsonb), ed25519_private_key_base64, COALESCE(jwk_pub,'{}'::jsonb) FROM trust_keys WHERE org_id=$1 AND active=true ORDER BY created_at DESC LIMIT 1`)).
		WithArgs("org_hs").
		WillReturnError(sqlmock.ErrCancelled)

	// Ensure HS256 secret is set; clear Ed25519
	_ = os.Unsetenv("AURA_TRUST_ED25519_PRIVATE_KEY")
	os.Setenv("JWT_SECRET", "hs_secret_test")
	defer os.Unsetenv("JWT_SECRET")

	r := setupRouter()
	w := httptest.NewRecorder()
	reqBody := `{"org_id":"org_hs","sub":"user_1","aud":"svc","action":"read","resource":"doc:1","ttl_sec":60}`
	req := httptest.NewRequest(http.MethodPost, "/issue", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("issue status: %d body=%s", w.Code, w.Body.String())
	}
	var out issueResp
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if out.Alg != "HS256" {
		t.Fatalf("expected HS256, got %s", out.Alg)
	}
	if out.Token == "" {
		t.Fatalf("empty token")
	}
	// Verify via handler
	w2 := httptest.NewRecorder()
	vr := map[string]string{"token": out.Token}
	vb, _ := json.Marshal(vr)
	req2 := httptest.NewRequest(http.MethodPost, "/verify", strings.NewReader(string(vb)))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("verify status: %d body=%s", w2.Code, w2.Body.String())
	}
	var vout verifyResp
	if err := json.Unmarshal(w2.Body.Bytes(), &vout); err != nil {
		t.Fatalf("json verify: %v", err)
	}
	if !vout.Valid {
		t.Fatalf("expected valid token; reason=%s", vout.Reason)
	}
	if vout.Claims["org_id"] != "org_hs" {
		t.Fatalf("org_id mismatch: %v", vout.Claims["org_id"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock unmet: %v", err)
	}
}

func TestIssueTrustTokenV1_EdDSAEnv(t *testing.T) {
	// Mock DB: no org key
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	database.DB = sqlx.NewDb(db, "sqlmock")
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT alg, COALESCE(kid,''), provider, key_ref, key_version, COALESCE(provider_config,'{}'::jsonb), ed25519_private_key_base64, COALESCE(jwk_pub,'{}'::jsonb) FROM trust_keys WHERE org_id=$1 AND active=true ORDER BY created_at DESC LIMIT 1`)).
		WithArgs("org_ed").
		WillReturnError(sqlmock.ErrCancelled)

	// Generate env Ed25519 seed
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519: %v", err)
	}
	// Use 32-byte seed part of private key
	seed := []byte(priv[:32])
	os.Setenv("AURA_TRUST_ED25519_PRIVATE_KEY", base64.RawURLEncoding.EncodeToString(seed))
	defer os.Unsetenv("AURA_TRUST_ED25519_PRIVATE_KEY")
	_ = os.Unsetenv("JWT_SECRET")

	r := setupRouter()
	w := httptest.NewRecorder()
	reqBody := `{"org_id":"org_ed","sub":"user_2","aud":"svc","action":"write","resource":"doc:2","ttl_sec":60}`
	req := httptest.NewRequest(http.MethodPost, "/issue", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("issue status: %d body=%s", w.Code, w.Body.String())
	}
	var out issueResp
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if out.Alg != "EdDSA" {
		t.Fatalf("expected EdDSA, got %s", out.Alg)
	}
	if out.Token == "" {
		t.Fatalf("empty token")
	}
	// Verify via handler
	w2 := httptest.NewRecorder()
	vr := map[string]string{"token": out.Token}
	vb, _ := json.Marshal(vr)
	req2 := httptest.NewRequest(http.MethodPost, "/verify", strings.NewReader(string(vb)))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("verify status: %d body=%s", w2.Code, w2.Body.String())
	}
	var vout verifyResp
	if err := json.Unmarshal(w2.Body.Bytes(), &vout); err != nil {
		t.Fatalf("json verify: %v", err)
	}
	if !vout.Valid {
		t.Fatalf("expected valid token; reason=%s", vout.Reason)
	}
	if vout.Claims["org_id"] != "org_ed" {
		t.Fatalf("org_id mismatch: %v", vout.Claims["org_id"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock unmet: %v", err)
	}
}

// no helpers
