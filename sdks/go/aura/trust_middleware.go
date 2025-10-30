package aura

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// TrustTokenMiddleware verifies a Bearer trust token offline using the provided TrustCache and writes 401 on failure.
// On success, it sets the claims JSON into request context under key "auraClaims" via WithContext.
func TrustTokenMiddleware(baseURL, orgId string, cache *TrustCache, graceSeconds int) func(http.Handler) http.Handler {
	if cache == nil {
		cache = NewTrustCache(5*time.Minute, 1*time.Minute)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing_token"})
				return
			}
			var token string
			if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
				token = strings.TrimSpace(auth[7:])
			} else {
				token = auth
			}
			res, err := VerifyTrustTokenOfflineCached(r.Context(), cache, baseURL, token, orgId, graceSeconds)
			if err != nil || !res.Valid {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				reason := "verify_error"
				if err == nil && !res.Valid {
					reason = res.Reason
				}
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_token", "reason": reason})
				return
			}
			// attach claims to context
			ctx := context.WithValue(r.Context(), struct{ k string }{"auraClaims"}, res.Claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
