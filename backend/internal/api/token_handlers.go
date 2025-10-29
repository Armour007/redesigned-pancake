package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	database "github.com/Armour007/aura-backend/internal"
)

type introspectReq struct {
	Token    string `json:"token"`
	MarkUsed bool   `json:"mark_used,omitempty"`
}

type introspectResp struct {
	Valid  bool        `json:"valid"`
	Reason string      `json:"reason,omitempty"`
	Claims interface{} `json:"claims,omitempty"`
}

// POST /v2/tokens/introspect
// Validates HS256 trust tokens issued by verify v2 and returns claims if valid
func IntrospectTrustToken(c *gin.Context) {
	var req introspectReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Token) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token required"})
		return
	}
	valid, claims, reason := validateJWT(req.Token)
	if !valid {
		c.JSON(http.StatusOK, introspectResp{Valid: false, Reason: reason})
		return
	}
	// Optional JTI replay prevention when mark_used=true
	if req.MarkUsed {
		// claims should include org_id (string), jti (string), exp (number)
		orgID, _ := claims["org_id"].(string)
		jti, _ := claims["jti"].(string)
		var expUnix int64
		if ev, ok := claims["exp"]; ok {
			switch v := ev.(type) {
			case float64:
				expUnix = int64(v)
			case json.Number:
				if n, err := v.Int64(); err == nil {
					expUnix = n
				}
			}
		}
		if orgID == "" || jti == "" || expUnix == 0 {
			c.JSON(http.StatusOK, introspectResp{Valid: false, Reason: "missing org_id/jti/exp"})
			return
		}
		// try to insert; if conflict, mark as replay
		res, err := database.DB.Exec(`INSERT INTO trust_token_jti(org_id, jti, exp_at) VALUES ($1,$2, to_timestamp($3)) ON CONFLICT DO NOTHING`, orgID, jti, expUnix)
		if err == nil {
			if n, _ := res.RowsAffected(); n == 0 {
				// already present -> replay
				c.JSON(http.StatusOK, introspectResp{Valid: false, Reason: "replayed"})
				return
			}
		}
	}
	c.JSON(http.StatusOK, introspectResp{Valid: true, Claims: claims})
}

// validateHS256JWT verifies signature and exp claim for compact JWT using env secret
func validateHS256JWT(tok string) (bool, map[string]any, string) {
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		return false, nil, "invalid token format"
	}
	headerB, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false, nil, "invalid header"
	}
	var hdr map[string]any
	if err := json.Unmarshal(headerB, &hdr); err != nil {
		return false, nil, "invalid header json"
	}
	if alg, _ := hdr["alg"].(string); alg != "HS256" {
		return false, nil, "unsupported alg"
	}
	secret := os.Getenv("AURA_TRUST_TOKEN_SIGNING_KEY")
	if secret == "" {
		secret = os.Getenv("JWT_SECRET")
	}
	if secret == "" {
		return false, nil, "signing key not configured"
	}
	unsigned := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	sig := mac.Sum(nil)
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)
	if !hmac.Equal([]byte(sigB64), []byte(parts[2])) {
		return false, nil, "invalid signature"
	}
	payloadB, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false, nil, "invalid payload"
	}
	var claims map[string]any
	if err := json.Unmarshal(payloadB, &claims); err != nil {
		return false, nil, "invalid payload json"
	}
	// check exp
	if expV, ok := claims["exp"]; ok {
		switch v := expV.(type) {
		case float64:
			if time.Now().Unix() > int64(v) {
				return false, nil, "token expired"
			}
		case json.Number:
			if n, err := v.Int64(); err == nil && time.Now().Unix() > n {
				return false, nil, "token expired"
			}
		default:
			// treat as invalid exp type
			return false, nil, "invalid exp"
		}
	} else {
		return false, nil, "missing exp"
	}
	return true, claims, ""
}

// validateJWT supports HS256 and EdDSA (Ed25519) based on header.alg
func validateJWT(tok string) (bool, map[string]any, string) {
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		return false, nil, "invalid token format"
	}
	headerB, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false, nil, "invalid header"
	}
	var hdr map[string]any
	if err := json.Unmarshal(headerB, &hdr); err != nil {
		return false, nil, "invalid header json"
	}
	alg, _ := hdr["alg"].(string)
	switch alg {
	case "HS256":
		return validateHS256JWT(tok)
	case "EdDSA":
		unsigned := parts[0] + "." + parts[1]
		kid, _ := hdr["kid"].(string)
		if !verifyEdDSA(unsigned, parts[2], kid) {
			return false, nil, "invalid signature"
		}
		payloadB, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return false, nil, "invalid payload"
		}
		var claims map[string]any
		if err := json.Unmarshal(payloadB, &claims); err != nil {
			return false, nil, "invalid payload json"
		}
		if expV, ok := claims["exp"]; ok {
			switch v := expV.(type) {
			case float64:
				if time.Now().Unix() > int64(v) {
					return false, nil, "token expired"
				}
			case json.Number:
				if n, err := v.Int64(); err == nil && time.Now().Unix() > n {
					return false, nil, "token expired"
				}
			default:
				return false, nil, "invalid exp"
			}
		} else {
			return false, nil, "missing exp"
		}
		return true, claims, ""
	default:
		return false, nil, "unsupported alg"
	}
}
