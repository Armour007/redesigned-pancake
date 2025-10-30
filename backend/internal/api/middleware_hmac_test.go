package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func signBody(secret, method, path, ts, nonce string, body []byte) string {
	unsigned := method + "\n" + path + "\n" + ts + "\n" + nonce + "\n" + string(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestRequestSigningMiddleware_OK(t *testing.T) {
	t.Setenv("AURA_REQUEST_HMAC_SECRET", "testsecret")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestSigningMiddleware())
	r.POST("/v1/test", func(c *gin.Context) { c.Status(200) })

	ts := time.Now().Unix()
	nonce := "n123"
	body := []byte(`{"a":1}`)
	sig := signBody("testsecret", http.MethodPost, "/v1/test",
		fmt.Sprintf("%d", ts), nonce, body)

	req := httptest.NewRequest(http.MethodPost, "/v1/test", bytes.NewReader(body))
	req.Header.Set("X-Aura-Timestamp", fmt.Sprintf("%d", ts))
	req.Header.Set("X-Aura-Nonce", nonce)
	req.Header.Set("X-Aura-Signature", sig)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRequestSigningMiddleware_BadSig(t *testing.T) {
	t.Setenv("AURA_REQUEST_HMAC_SECRET", "testsecret")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestSigningMiddleware())
	r.POST("/v1/test", func(c *gin.Context) { c.Status(200) })

	ts := time.Now().Unix()
	req := httptest.NewRequest(http.MethodPost, "/v1/test", bytes.NewReader([]byte("{}")))
	req.Header.Set("X-Aura-Timestamp", fmt.Sprintf("%d", ts))
	req.Header.Set("X-Aura-Nonce", "n1")
	req.Header.Set("X-Aura-Signature", "deadbeef")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
