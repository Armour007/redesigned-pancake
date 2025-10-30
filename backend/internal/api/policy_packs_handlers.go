package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type PolicyPack struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Industry    string           `json:"industry"`
	Description string           `json:"description"`
	Engine      string           `json:"engine"`
	Actions     []string         `json:"actions"`
	Body        map[string]any   `json:"body"`
	Tests       []map[string]any `json:"tests"`
}

// Minimal built-in packs registry (can be extended to pull from a catalog)
var policyPacks = []PolicyPack{
	{
		ID:          "soc2-base-aurajson",
		Name:        "SOC2 Base Controls",
		Industry:    "Any",
		Description: "Baseline guardrails aligned with SOC2 CC5/CC6: admin approval for sensitive actions, deny on missing posture.",
		Engine:      "aurajson",
		Actions:     []string{"issue_cert", "delete_agent", "rotate_trust_key"},
		Body: map[string]any{
			"rules": []any{
				map[string]any{"when": map[string]any{"action": "issue_cert", "device.posture_ok": true}, "allow": true, "reason": "device posture ok"},
				map[string]any{"when": map[string]any{"action": "delete_agent"}, "require_approval": true, "reason": "sensitive deletion"},
				map[string]any{"when": map[string]any{"action": "rotate_trust_key"}, "allow": true},
			},
			"default": map[string]any{"deny": true, "reason": "not permitted"},
		},
		Tests: []map[string]any{
			{"name": "issue_cert posture ok", "input": map[string]any{"action": "issue_cert", "device": map[string]any{"posture_ok": true}}, "expect": map[string]any{"allow": true}},
			{"name": "delete_agent requires approval", "input": map[string]any{"action": "delete_agent"}, "expect": map[string]any{"require_approval": true}},
		},
	},
	{
		ID:          "gdpr-data-export-guard",
		Name:        "GDPR Data Export Guard",
		Industry:    "Any",
		Description: "Require explicit approval for PII export and restrict cross-region unless whitelisted.",
		Engine:      "aurajson",
		Actions:     []string{"export_pii"},
		Body: map[string]any{
			"rules": []any{
				map[string]any{"when": map[string]any{"action": "export_pii", "region": []any{"EU", "EEA"}}, "allow": true},
				map[string]any{"when": map[string]any{"action": "export_pii"}, "require_approval": true, "reason": "cross-region export"},
			},
			"default": map[string]any{"deny": true},
		},
		Tests: []map[string]any{
			{"name": "eu export allowed", "input": map[string]any{"action": "export_pii", "region": "EU"}, "expect": map[string]any{"allow": true}},
		},
	},
}

// GET /v2/policy/packs
func ListPolicyPacks(c *gin.Context) {
	items := make([]gin.H, 0, len(policyPacks))
	for _, p := range policyPacks {
		items = append(items, gin.H{"id": p.ID, "name": p.Name, "industry": p.Industry, "description": p.Description, "engine": p.Engine, "actions": p.Actions})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// GET /v2/policy/packs/:packId
func GetPolicyPack(c *gin.Context) {
	id := c.Param("packId")
	for _, p := range policyPacks {
		if p.ID == id {
			c.JSON(http.StatusOK, p)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
}
