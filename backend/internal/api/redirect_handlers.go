package api

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// QuickstartRedirect issues a 302 to the frontend Quick Start page with the
// provided query parameters. This enables simple linking from backend flows
// (or Swagger) to the first-time SDK guide without building a full dashboard yet.
//
// Usage:
//
//	GET /quickstart?agent_id=...&key_prefix=...
//
// Env:
//
//	AURA_FRONTEND_BASE_URL or PUBLIC_SITE_URL -> e.g., http://localhost:5173
func QuickstartRedirect(c *gin.Context) {
	agentID := c.Query("agent_id")
	keyPrefix := c.Query("key_prefix")

	frontendBase := strings.TrimRight(func() string {
		if v := os.Getenv("AURA_FRONTEND_BASE_URL"); v != "" {
			return v
		}
		if v := os.Getenv("PUBLIC_SITE_URL"); v != "" {
			return v
		}
		return "http://localhost:5173"
	}(), "/")

	// Build target URL safely with escaping
	q := url.Values{}
	if agentID != "" {
		q.Set("agent_id", agentID)
	}
	if keyPrefix != "" {
		q.Set("key_prefix", keyPrefix)
	}
	target := frontendBase + "/quickstart"
	if enc := q.Encode(); enc != "" {
		target = target + "?" + enc
	}
	c.Redirect(http.StatusFound, target)
}
