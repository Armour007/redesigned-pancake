package api

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	database "github.com/Armour007/aura-backend/internal"
	redis "github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
)

// getRedisFromEnv returns a Redis client if configured via AURA_REDIS_ADDR, else nil
func getRedisFromEnvLocal() *redis.Client {
	addr := os.Getenv("AURA_REDIS_ADDR")
	if addr == "" {
		return nil
	}
	return redis.NewClient(&redis.Options{Addr: addr, Password: os.Getenv("AURA_REDIS_PASSWORD")})
}

type jwk struct {
	Kty string `json:"kty"`
	Crv string `json:"crv,omitempty"`
	Alg string `json:"alg,omitempty"`
	Use string `json:"use,omitempty"`
	Kid string `json:"kid,omitempty"`
	X   string `json:"x,omitempty"`
	Y   string `json:"y,omitempty"`
}

type jwks struct {
	Keys []jwk `json:"keys"`
}

// loadEd25519KeyFromEnv reads AURA_TRUST_ED25519_PRIVATE_KEY (base64url or base64) and returns priv, pub, kid.
func loadEd25519KeyFromEnv() (ed25519.PrivateKey, ed25519.PublicKey, string) {
	enc := os.Getenv("AURA_TRUST_ED25519_PRIVATE_KEY")
	if enc == "" {
		return nil, nil, ""
	}
	// support both RawURLEncoding and StdEncoding
	var raw []byte
	if b, err := base64.RawURLEncoding.DecodeString(enc); err == nil {
		raw = b
	} else if b, err2 := base64.StdEncoding.DecodeString(enc); err2 == nil {
		raw = b
	} else {
		return nil, nil, ""
	}
	var priv ed25519.PrivateKey
	switch len(raw) {
	case ed25519.SeedSize:
		priv = ed25519.NewKeyFromSeed(raw)
	case ed25519.PrivateKeySize:
		priv = ed25519.PrivateKey(raw)
	default:
		return nil, nil, ""
	}
	pub := priv.Public().(ed25519.PublicKey)
	kid := os.Getenv("AURA_TRUST_KEY_ID")
	if kid == "" {
		sum := sha256.Sum256(pub)
		kid = base64.RawURLEncoding.EncodeToString(sum[:8]) // short kid
	}
	return priv, pub, kid
}

// loadOrgEd25519Key returns the active org-specific Ed25519 key if present in DB
// Falls back to nils if not found or not EdDSA.
func loadOrgEd25519Key(orgID string) (ed25519.PrivateKey, ed25519.PublicKey, string) {
	if strings.TrimSpace(orgID) == "" {
		return nil, nil, ""
	}
	var row struct {
		Alg string  `db:"alg"`
		Key string  `db:"ed25519_private_key_base64"`
		Kid *string `db:"kid"`
	}
	if err := database.DB.Get(&row, `SELECT alg, ed25519_private_key_base64, kid FROM trust_keys WHERE org_id=$1 AND active=true ORDER BY created_at DESC LIMIT 1`, orgID); err != nil {
		return nil, nil, ""
	}
	if strings.ToUpper(row.Alg) != "EDDSA" || row.Key == "" {
		return nil, nil, ""
	}
	var raw []byte
	if b, err := base64.RawURLEncoding.DecodeString(row.Key); err == nil {
		raw = b
	} else if b, err2 := base64.StdEncoding.DecodeString(row.Key); err2 == nil {
		raw = b
	} else {
		return nil, nil, ""
	}
	var priv ed25519.PrivateKey
	switch len(raw) {
	case ed25519.SeedSize:
		priv = ed25519.NewKeyFromSeed(raw)
	case ed25519.PrivateKeySize:
		priv = ed25519.PrivateKey(raw)
	default:
		return nil, nil, ""
	}
	pub := priv.Public().(ed25519.PublicKey)
	kid := ""
	if row.Kid != nil {
		kid = *row.Kid
	}
	if kid == "" {
		sum := sha256.Sum256(pub)
		kid = base64.RawURLEncoding.EncodeToString(sum[:8])
	}
	return priv, pub, kid
}

// JWKS publishes the public signing key used for trust tokens (if configured)
func JWKS(c *gin.Context) {
	baseCtx := context.Background()
	if c.Request != nil {
		baseCtx = c.Request.Context()
	}
	ctx, span := otel.Tracer("aura-backend").Start(baseCtx, "jwks.global")
	defer span.End()
	// Try Redis cache first
	rc := getRedisFromEnvLocal()
	if rc != nil {
		ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		defer cancel()
		if b, err := rc.Get(ctx, "jwks:global").Bytes(); err == nil && len(b) > 0 {
			RecordCacheHit("jwks", "global")
			c.Data(200, "application/json", b)
			return
		}
		RecordCacheMiss("jwks", "global")
	}
	_, pub, kid := loadEd25519KeyFromEnv()
	if pub == nil {
		c.JSON(200, jwks{Keys: []jwk{}})
		return
	}
	x := base64.RawURLEncoding.EncodeToString(pub)
	k := jwk{Kty: "OKP", Crv: "Ed25519", Alg: "EdDSA", Use: "sig", Kid: kid, X: x}
	payload := jwks{Keys: []jwk{k}}
	if rc != nil {
		ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		defer cancel()
		if jb, err := json.Marshal(payload); err == nil {
			ttl := 10 * time.Minute
			if v := os.Getenv("AURA_JWKS_CACHE_TTL_SECONDS"); v != "" {
				if n, err2 := strconv.Atoi(v); err2 == nil && n > 0 {
					ttl = time.Duration(n) * time.Second
				}
			}
			_ = rc.Set(ctx, "jwks:global", jb, ttl).Err()
		}
	}
	c.JSON(200, payload)
}

// OrgJWKS publishes org-scoped JWKS with all active keys for rotation overlap
// GET /.well-known/aura/:orgId/jwks.json
func OrgJWKS(c *gin.Context) {
	orgID := c.Param("orgId")
	baseCtx := context.Background()
	if c.Request != nil {
		baseCtx = c.Request.Context()
	}
	ctx, span := otel.Tracer("aura-backend").Start(baseCtx, "jwks.org")
	defer span.End()
	// Redis cached response
	rc := getRedisFromEnvLocal()
	if rc != nil {
		ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		defer cancel()
		key := fmt.Sprintf("jwks:org:%s", orgID)
		if b, err := rc.Get(ctx, key).Bytes(); err == nil && len(b) > 0 {
			RecordCacheHit("jwks", "org")
			c.Data(200, "application/json", b)
			return
		}
		RecordCacheMiss("jwks", "org")
	}
	rows, err := database.DB.Queryx(`SELECT alg, COALESCE(jwk_pub,'{}'::jsonb), COALESCE(ed25519_private_key_base64,''), COALESCE(kid,'') FROM trust_keys WHERE org_id=$1 AND active=true ORDER BY created_at DESC LIMIT 10`, orgID)
	if err != nil {
		c.JSON(200, jwks{Keys: []jwk{}})
		return
	}
	defer rows.Close()
	keys := []jwk{}
	for rows.Next() {
		var alg string
		var jwkJSON json.RawMessage
		var enc string
		var kid string
		if err := rows.Scan(&alg, &jwkJSON, &enc, &kid); err != nil {
			continue
		}
		// Prefer stored public JWK when present
		if len(jwkJSON) > 0 && string(jwkJSON) != "{}" {
			var m map[string]any
			if err := json.Unmarshal(jwkJSON, &m); err == nil {
				// Compute kid if empty for Ed25519 from x
				if kid == "" && fmt.Sprint(m["kty"]) == "OKP" {
					if xStr, _ := m["x"].(string); xStr != "" {
						if xb, err := base64.RawURLEncoding.DecodeString(xStr); err == nil {
							sum := sha256.Sum256(xb)
							kid = base64.RawURLEncoding.EncodeToString(sum[:8])
						}
					}
				}
				if kid != "" {
					m["kid"] = kid
				}
				if fmt.Sprint(m["kty"]) == "OKP" {
					keys = append(keys, jwk{Kty: "OKP", Crv: fmt.Sprint(m["crv"]), Alg: fmt.Sprint(m["alg"]), Use: "sig", Kid: kid, X: fmt.Sprint(m["x"])})
					continue
				} else if fmt.Sprint(m["kty"]) == "EC" {
					keys = append(keys, jwk{Kty: "EC", Crv: fmt.Sprint(m["crv"]), Alg: fmt.Sprint(m["alg"]), Use: "sig", Kid: kid, X: fmt.Sprint(m["x"]), Y: fmt.Sprint(m["y"])})
					continue
				}
			}
		}
		// Fallback: derive EdDSA JWK from local private key if available
		if strings.ToUpper(alg) == "EDDSA" && enc != "" {
			var raw []byte
			if b, err := base64.RawURLEncoding.DecodeString(enc); err == nil {
				raw = b
			} else if b, err2 := base64.StdEncoding.DecodeString(enc); err2 == nil {
				raw = b
			} else {
				continue
			}
			var priv ed25519.PrivateKey
			switch len(raw) {
			case ed25519.SeedSize:
				priv = ed25519.NewKeyFromSeed(raw)
			case ed25519.PrivateKeySize:
				priv = ed25519.PrivateKey(raw)
			default:
				continue
			}
			pub := priv.Public().(ed25519.PublicKey)
			if kid == "" {
				sum := sha256.Sum256(pub)
				kid = base64.RawURLEncoding.EncodeToString(sum[:8])
			}
			x := base64.RawURLEncoding.EncodeToString(pub)
			keys = append(keys, jwk{Kty: "OKP", Crv: "Ed25519", Alg: "EdDSA", Use: "sig", Kid: kid, X: x})
		}
	}
	payload := jwks{Keys: keys}
	if rc != nil {
		ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		defer cancel()
		if jb, err := json.Marshal(payload); err == nil {
			ttl := 10 * time.Minute
			if v := os.Getenv("AURA_JWKS_CACHE_TTL_SECONDS"); v != "" {
				if n, err2 := strconv.Atoi(v); err2 == nil && n > 0 {
					ttl = time.Duration(n) * time.Second
				}
			}
			_ = rc.Set(ctx, fmt.Sprintf("jwks:org:%s", orgID), jb, ttl).Err()
		}
	}
	c.JSON(200, payload)
}

// verifyEdDSA verifies a compact JWS (header.payload.signature) using kid to locate the public key
// If kid is empty, falls back to env-configured public key.
func verifyEdDSA(unsigned string, sigB64 string, kid string) bool {
	pub, _ := findEd25519PubByKid(kid)
	if pub == nil {
		return false
	}
	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return false
	}
	return ed25519.Verify(pub, []byte(unsigned), sig)
}

// findEd25519PubByKid returns the pub key by kid; if kid empty, returns env pub if available
func findEd25519PubByKid(kid string) (ed25519.PublicKey, error) {
	// if kid empty, try env
	if strings.TrimSpace(kid) == "" {
		_, pub, _ := loadEd25519KeyFromEnv()
		if pub == nil {
			return nil, errors.New("no key available")
		}
		return pub, nil
	}
	// check env kid first
	if _, envPub, envKid := loadEd25519KeyFromEnv(); envPub != nil && envKid == kid {
		return envPub, nil
	}
	// query DB by kid (prefer jwk_pub)
	var out struct {
		JWK json.RawMessage `db:"jwk_pub"`
		Key string          `db:"ed25519_private_key_base64"`
	}
	if err := database.DB.Get(&out, `SELECT jwk_pub, COALESCE(ed25519_private_key_base64,'') FROM trust_keys WHERE kid=$1 AND active=true ORDER BY created_at DESC LIMIT 1`, kid); err != nil {
		return nil, errors.New("kid not found")
	}
	if len(out.JWK) > 0 && string(out.JWK) != "{}" {
		var m map[string]any
		if err := json.Unmarshal(out.JWK, &m); err == nil {
			if x, _ := m["x"].(string); x != "" {
				if b, err := base64.RawURLEncoding.DecodeString(x); err == nil && len(b) == ed25519.PublicKeySize {
					return ed25519.PublicKey(b), nil
				}
			}
		}
	}
	if out.Key == "" {
		return nil, errors.New("kid not found")
	}
	var raw []byte
	if b, err := base64.RawURLEncoding.DecodeString(out.Key); err == nil {
		raw = b
	} else if b, err2 := base64.StdEncoding.DecodeString(out.Key); err2 == nil {
		raw = b
	} else {
		return nil, errors.New("invalid key encoding")
	}
	var priv ed25519.PrivateKey
	switch len(raw) {
	case ed25519.SeedSize:
		priv = ed25519.NewKeyFromSeed(raw)
	case ed25519.PrivateKeySize:
		priv = ed25519.PrivateKey(raw)
	default:
		return nil, errors.New("invalid key length")
	}
	pub := priv.Public().(ed25519.PublicKey)
	return pub, nil
}

// helper to detect base64url string
func looksBase64URL(s string) bool {
	return !strings.ContainsAny(s, "+/=")
}
