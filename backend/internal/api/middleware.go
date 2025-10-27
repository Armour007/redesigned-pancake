package api

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// getJwtSecret retrieves the JWT secret from environment variables (reuse from jwt.go logic)
func getJwtSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable not set")
	}
	return []byte(secret), nil
}

// AuthMiddleware creates a Gin middleware for JWT authentication
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		// Check if the header format is "Bearer token"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			return
		}

		tokenString := parts[1]

		// Get the secret key
		jwtSecret, err := getJwtSecret()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "JWT secret configuration error"})
			return
		}

		// Parse and validate the token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Ensure the signing method is HMAC
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})

		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: " + err.Error()})
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			userID := claims["user_id"].(string)
			c.Set("userID", userID)
			c.Next()
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		}
	}
}

// OrgMemberMiddleware ensures the JWT user belongs to the organization in the :orgId route param.
func OrgMemberMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := c.Get("userID")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		orgID := c.Param("orgId")
		if orgID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing organization id"})
			return
		}
		var role string
		err := database.DB.Get(&role, `SELECT role FROM organization_members WHERE organization_id=$1 AND user_id=$2`, orgID, userID)
		if err != nil || role == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Not a member of this organization"})
			return
		}
		c.Set("orgRole", role)
		c.Next()
	}
}

// ApiKeyAuthMiddleware authenticates requests using an Organization API key.
// Expected header: either X-API-Key or Authorization: AURA <key>
// On success, sets orgID, apiKeyPrefix, apiKeyID in context.
func ApiKeyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("X-API-Key")
		if raw == "" {
			auth := c.GetHeader("Authorization")
			if auth != "" {
				parts := strings.SplitN(auth, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "AURA") {
					raw = parts[1]
				}
			}
		}
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			return
		}
		const prefix = "aura_sk_"
		if !strings.HasPrefix(raw, prefix) || len(raw) <= len(prefix)+8 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key format"})
			return
		}
		randomPart := raw[len(prefix):]
		keyPrefix := randomPart[:8]

		var key database.APIKey
		err := database.DB.Get(&key, `SELECT id, organization_id, name, key_prefix, hashed_key, created_by_user_id, last_used_at, expires_at, created_at FROM api_keys WHERE key_prefix=$1 AND revoked_at IS NULL LIMIT 1`, keyPrefix)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API key not found or revoked"})
			return
		}
		if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API key expired"})
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(key.HashedKey), []byte(raw)); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			return
		}
		now := time.Now()
		_, _ = database.DB.Exec(`UPDATE api_keys SET last_used_at=$1 WHERE id=$2`, now, key.ID)
		c.Set("orgID", key.OrganizationID.String())
		c.Set("apiKeyPrefix", keyPrefix)
		c.Set("apiKeyID", key.ID.String())
		c.Next()
	}
}

// RequestIDMiddleware ensures every request has an X-Request-ID. If absent, generate one.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = uuid.New().String()
		}
		ctx := context.WithValue(c.Request.Context(), "requestID", rid)
		c.Request = c.Request.WithContext(ctx)
		c.Set("requestID", rid)
		c.Writer.Header().Set("X-Request-ID", rid)
		c.Next()
	}
}

// RequireOrgAdmin checks that the authenticated org member is admin or owner.
func RequireOrgAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("orgRole")
		if role == "admin" || role == "owner" {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
	}
}

// Simple in-memory IP rate limiter (fixed window)
type clientWindow struct {
	count       int
	windowStart time.Time
}

type ipLimiter struct {
	mu      sync.Mutex
	clients map[string]*clientWindow
	limit   int
	window  time.Duration
}

func newIPLimiter(limit int, window time.Duration) *ipLimiter {
	return &ipLimiter{
		clients: make(map[string]*clientWindow),
		limit:   limit,
		window:  window,
	}
}

func (l *ipLimiter) allow(ip string) (bool, time.Duration) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	cw, ok := l.clients[ip]
	if !ok {
		l.clients[ip] = &clientWindow{count: 1, windowStart: now}
		return true, 0
	}
	if now.Sub(cw.windowStart) >= l.window {
		cw.count = 1
		cw.windowStart = now
		return true, 0
	}
	if cw.count < l.limit {
		cw.count++
		return true, 0
	}
	retryAfter := l.window - now.Sub(cw.windowStart)
	return false, retryAfter
}

// RateLimitMiddleware limits requests per client IP. Intended for /v1 endpoints.
func RateLimitMiddleware(limitPerMinute int) gin.HandlerFunc {
	if limitPerMinute <= 0 {
		limitPerMinute = 60
	}
	limiter := newIPLimiter(limitPerMinute, time.Minute)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		// if behind proxy and gin isn't configured with TrustedProxies, c.ClientIP uses X-Forwarded-For when safe.
		if net.ParseIP(ip) == nil {
			ip = "unknown"
		}
		ok, retryAfter := limiter.allow(ip)
		if !ok {
			c.Header("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded. Try again later."})
			return
		}
		c.Next()
	}
}

// --- Optional Redis-backed rate limiter for distributed deployments ---
// Uses minute-window keys per client IP. Enable with AURA_REDIS_ADDR; configure limit via AURA_V1_VERIFY_RPM.
// Falls back to in-memory limiter if Redis is not configured.

// RateLimitMiddlewareFromEnv builds a rate-limit middleware using env config.
// AURA_V1_VERIFY_RPM (default 60). If AURA_REDIS_ADDR is set, use Redis; else in-memory.
func RateLimitMiddlewareFromEnv() gin.HandlerFunc {
	rpm := 60
	if v := os.Getenv("AURA_V1_VERIFY_RPM"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rpm = n
		}
	}
	addr := os.Getenv("AURA_REDIS_ADDR")
	if addr == "" {
		return RateLimitMiddleware(rpm)
	}
	rc := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("AURA_REDIS_PASSWORD"),
		DB:       parseEnvInt("AURA_REDIS_DB", 0),
	})

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if net.ParseIP(ip) == nil {
			ip = "unknown"
		}
		now := time.Now().UTC()
		key := fmt.Sprintf("rl:%s:%04d%02d%02d%02d%02d", ip, now.Year(), int(now.Month()), now.Day(), now.Hour(), now.Minute())
		ctx, cancel := context.WithTimeout(c.Request.Context(), 200*time.Millisecond)
		defer cancel()

		if err := rc.Ping(ctx).Err(); err != nil {
			RateLimitMiddleware(rpm)(c)
			return
		}
		n, err := rc.Incr(ctx, key).Result()
		if err != nil {
			RateLimitMiddleware(rpm)(c)
			return
		}
		_ = rc.Expire(ctx, key, 61*time.Second).Err()
		if int(n) > rpm {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded. Try again later."})
			return
		}
		c.Next()
	}
}

// helpers
func parseEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// --- API Version middleware ---
// Reads AURA-Version request header; if absent, uses default; always sets X-AURA-Version in response.
func VersionMiddleware(defaultVersion string) gin.HandlerFunc {
	if defaultVersion == "" {
		defaultVersion = "2025-10-01"
	}
	return func(c *gin.Context) {
		ver := c.GetHeader("AURA-Version")
		if ver == "" {
			ver = defaultVersion
		}
		c.Set("auraVersion", ver)
		c.Writer.Header().Set("X-AURA-Version", ver)
		c.Next()
	}
}

// --- Idempotency middleware (Redis-backed if configured, else in-memory) ---
type captureWriter struct {
	gin.ResponseWriter
	status int
	buf    bytes.Buffer
}

func (w *captureWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
func (w *captureWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.buf.Write(b)
	return w.ResponseWriter.Write(b)
}

var (
	idemStore sync.Map // key -> struct{status int, body []byte, ts time.Time}
)

func getRedisFromEnv() *redis.Client {
	addr := os.Getenv("AURA_REDIS_ADDR")
	if addr == "" {
		return nil
	}
	return redis.NewClient(&redis.Options{Addr: addr, Password: os.Getenv("AURA_REDIS_PASSWORD"), DB: parseEnvInt("AURA_REDIS_DB", 0)})
}

// IdempotencyMiddlewareFromEnv caches responses for POST requests that include Idempotency-Key header
// Applies to typical mutating org routes: /organizations/:orgId/(agents|apikeys|agents/:id/permissions)
func IdempotencyMiddlewareFromEnv() gin.HandlerFunc {
	rc := getRedisFromEnv()
	ttl := time.Hour * 24
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}
		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			c.Next()
			return
		}
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		if !strings.Contains(path, "/organizations/") {
			c.Next()
			return
		}
		// scope with org id when available
		orgID := c.Param("orgId")
		storageKey := fmt.Sprintf("idem:%s:%s", orgID, key)
		// Redis check
		if rc != nil {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 250*time.Millisecond)
			defer cancel()
			if data, err := rc.Get(ctx, storageKey).Bytes(); err == nil && len(data) >= 3 {
				// first 3 bytes encode status as text? keep it simple: status and a pipe separator
				// format: <status>\n<body>
				// find first newline
				for i := 0; i < len(data); i++ {
					if data[i] == '\n' {
						statusStr := string(data[:i])
						body := data[i+1:]
						if s, err2 := strconv.Atoi(statusStr); err2 == nil {
							c.Status(s)
							c.Writer.Header().Set("X-Idempotent-Replay", "true")
							_, _ = c.Writer.Write(body)
							c.Abort()
							return
						}
						break
					}
				}
			}
		} else {
			if v, ok := idemStore.Load(storageKey); ok {
				if rec, ok2 := v.(struct {
					status int
					body   []byte
					ts     time.Time
				}); ok2 {
					c.Status(rec.status)
					c.Writer.Header().Set("X-Idempotent-Replay", "true")
					_, _ = c.Writer.Write(rec.body)
					c.Abort()
					return
				}
			}
		}

		cw := &captureWriter{ResponseWriter: c.Writer}
		c.Writer = cw
		c.Next()
		// after handler, store result
		status := cw.status
		body := cw.buf.Bytes()
		if rc != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
			defer cancel()
			payload := []byte(strconv.Itoa(status) + "\n")
			payload = append(payload, body...)
			_ = rc.Set(ctx, storageKey, payload, ttl).Err()
		} else {
			idemStore.Store(storageKey, struct {
				status int
				body   []byte
				ts     time.Time
			}{status: status, body: append([]byte(nil), body...), ts: time.Now()})
		}
	}
}
