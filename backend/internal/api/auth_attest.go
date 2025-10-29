package api

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AttestRequest struct {
	OrgID   string `json:"org_id"`
	AgentID string `json:"agent_id"`
}

type AttestResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

// Attest mints a short-lived attestation JWT based on SPIFFE identity headers.
// Guarded by AURA_ATTEST_ENABLE env var. Signing key: AURA_ATTEST_SIGNING_KEY (fallback JWT_SECRET).
func Attest(c *gin.Context) {
	if enabled := os.Getenv("AURA_ATTEST_ENABLE"); enabled == "" || enabled == "0" || enabled == "false" {
		c.JSON(http.StatusNotFound, gin.H{"error": "endpoint disabled"})
		return
	}
	spiffe := c.GetHeader("X-SPIFFE-ID")
	if spiffe == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing X-SPIFFE-ID header"})
		return
	}
	var req AttestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// TTL seconds (default 300s)
	ttl := 300 * time.Second
	if v := os.Getenv("AURA_ATTEST_TTL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			ttl = time.Duration(n) * time.Second
		}
	}
	now := time.Now()
	exp := now.Add(ttl)

	// Build claims
	claims := jwt.MapClaims{
		"sub":        spiffe,
		"iss":        "aura-backend",
		"aud":        "aura",
		"iat":        now.Unix(),
		"nbf":        now.Unix(),
		"exp":        exp.Unix(),
		"kind":       "attest",
		"org_id":     req.OrgID,
		"agent_id":   req.AgentID,
		"authn_kind": "spiffe",
	}

	key := []byte(os.Getenv("AURA_ATTEST_SIGNING_KEY"))
	if len(key) == 0 {
		// fallback to JWT_SECRET for convenience in dev
		key = []byte(os.Getenv("JWT_SECRET"))
	}
	if len(key) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no signing key configured"})
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString(key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign token"})
		return
	}
	c.JSON(http.StatusOK, AttestResponse{Token: s, ExpiresAt: exp.Unix()})
}
