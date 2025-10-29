package api

import (
	"net/http"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
)

// GetDecisionTrace returns stored decision trace by trace_id (org-scoped via API key)
func GetDecisionTrace(c *gin.Context) {
	orgID := c.GetString("orgID")
	traceID := c.Param("traceId")
	if traceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing trace id"})
		return
	}
	t := struct {
		OrgID         string  `db:"org_id" json:"org_id"`
		TraceID       string  `db:"trace_id" json:"trace_id"`
		PolicyID      *string `db:"policy_id" json:"policy_id"`
		PolicyVersion *int    `db:"policy_version" json:"policy_version"`
		AgentID       *string `db:"agent_id" json:"agent_id"`
		Allow         bool    `db:"allow" json:"allow"`
		Reason        *string `db:"reason" json:"reason"`
		Trace         any     `db:"trace" json:"trace"`
		CreatedAt     string  `db:"created_at" json:"created_at"`
	}{}
	if err := database.DB.Get(&t, `SELECT org_id, trace_id, policy_id::text, policy_version, agent_id::text, allow, reason, trace, created_at::text FROM decision_traces WHERE org_id=$1 AND trace_id=$2`, orgID, traceID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trace not found"})
		return
	}
	c.JSON(http.StatusOK, t)
}
