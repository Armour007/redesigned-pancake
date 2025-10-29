package api

import (
	"net/http"
	"strconv"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
)

// GetRecentDecisionTraces lists recent decision traces for the caller's org.
// GET /v2/decisions?limit=50&agent_id=&allow=
func GetRecentDecisionTraces(c *gin.Context) {
	orgID := c.GetString("orgID")
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			if n < 1 {
				n = 1
			}
			if n > 500 {
				n = 500
			}
			limit = n
		}
	}
	agentID := c.Query("agent_id")
	allow := c.Query("allow") // optional: "true" | "false"

	// Build simple filtered query
	type row struct {
		OrgID     string  `db:"org_id" json:"org_id"`
		TraceID   string  `db:"trace_id" json:"trace_id"`
		PolicyID  *string `db:"policy_id" json:"policy_id"`
		Version   *int    `db:"policy_version" json:"policy_version"`
		AgentID   *string `db:"agent_id" json:"agent_id"`
		Allow     bool    `db:"allow" json:"allow"`
		Reason    *string `db:"reason" json:"reason"`
		CreatedAt string  `db:"created_at" json:"created_at"`
	}

	rows := []row{}
	// Branch on filters to keep SQL and bindings straightforward
	switch {
	case agentID != "" && (allow == "true" || allow == "false"):
		_ = database.DB.Select(&rows, `
            SELECT org_id, trace_id, policy_id::text, policy_version, agent_id::text, allow, reason, created_at::text
            FROM decision_traces
            WHERE org_id=$1 AND agent_id::text=$2 AND allow=$3
            ORDER BY created_at DESC
            LIMIT $4
        `, orgID, agentID, allow == "true", limit)
	case agentID != "":
		_ = database.DB.Select(&rows, `
            SELECT org_id, trace_id, policy_id::text, policy_version, agent_id::text, allow, reason, created_at::text
            FROM decision_traces
            WHERE org_id=$1 AND agent_id::text=$2
            ORDER BY created_at DESC
            LIMIT $3
        `, orgID, agentID, limit)
	case allow == "true" || allow == "false":
		_ = database.DB.Select(&rows, `
            SELECT org_id, trace_id, policy_id::text, policy_version, agent_id::text, allow, reason, created_at::text
            FROM decision_traces
            WHERE org_id=$1 AND allow=$2
            ORDER BY created_at DESC
            LIMIT $3
        `, orgID, allow == "true", limit)
	default:
		_ = database.DB.Select(&rows, `
            SELECT org_id, trace_id, policy_id::text, policy_version, agent_id::text, allow, reason, created_at::text
            FROM decision_traces
            WHERE org_id=$1
            ORDER BY created_at DESC
            LIMIT $2
        `, orgID, limit)
	}

	c.JSON(http.StatusOK, gin.H{"org_id": orgID, "items": rows, "count": len(rows)})
}
