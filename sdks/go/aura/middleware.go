package aura

import (
	"encoding/json"
	"net/http"
)

// ProtectHTTP wraps an http.Handler and verifies with AURA before invoking next.
// contextFromReq builds the request_context payload; if nil, a default is used.
// onDeny is called when decision != ALLOWED; if nil, responds 403 with JSON error.
func ProtectHTTP(agentID string, client *Client, contextFromReq func(*http.Request) any, onDeny func(http.ResponseWriter, *http.Request, string)) func(http.Handler) http.Handler {
	if contextFromReq == nil {
		contextFromReq = func(r *http.Request) any {
			return map[string]any{
				"path":   r.URL.Path,
				"method": r.Method,
			}
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := contextFromReq(r)
			res, err := client.Verify(agentID, ctx)
			if err != nil || res == nil || res.Decision != "ALLOWED" {
				reason := "access denied"
				if res != nil && res.Reason != "" {
					reason = res.Reason
				}
				if onDeny != nil {
					onDeny(w, r, reason)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "AURA denied", "reason": reason})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
