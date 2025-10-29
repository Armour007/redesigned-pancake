package api

import (
	"net/http"
	"strconv"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
)

// GET /v2/policy/recommendations?org_id=&limit=200
// Prototype: summarize recent decision reasons and suggest textual hints (no automatic changes)
func GetPolicyRecommendations(c *gin.Context) {
	orgID := c.Query("org_id")
	if orgID == "" {
		orgID = c.GetString("orgID")
	}
	limit := 200
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 2000 {
			limit = n
		}
	}
	type row struct {
		Reason *string `db:"reason"`
		Cnt    int     `db:"cnt"`
	}
	rows := []row{}
	_ = database.DB.Select(&rows, `SELECT reason, COUNT(*) cnt FROM decision_traces WHERE org_id=$1 GROUP BY reason ORDER BY cnt DESC LIMIT $2`, orgID, limit)

	suggestions := []map[string]any{}
	for _, r := range rows {
		if r.Reason == nil {
			continue
		}
		reason := *r.Reason
		// Very naive mapping: if high risk frequently, suggest stricter deny rule; if deny for missing delegation, suggest adding tuples.
		if reason == "high risk" || reason == "rate_spike" {
			suggestions = append(suggestions, map[string]any{
				"type":         "policy_hint",
				"message":      "Consider lowering risk threshold or adding cooldown for agents with high risk score",
				"example_rule": map[string]any{"gte": []any{map[string]any{"var": "risk.score"}, 70}, "effect": "deny"},
			})
		}
		if reason == "No delegation to act for org" {
			suggestions = append(suggestions, map[string]any{
				"type":    "relationship_hint",
				"message": "Many checks fail due to missing delegation; review tuples granting can_act_for to agents",
			})
		}
	}
	c.JSON(http.StatusOK, gin.H{"org_id": orgID, "recommendations": suggestions, "reasons": rows})
}
