package api

import (
	"net/http"
	"time"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GET /v2/reputation?org_id=&agent_id=
func GetReputation(c *gin.Context) {
	orgID := c.Query("org_id")
	if orgID == "" {
		orgID = c.GetString("orgID")
	}
	agentID := c.Query("agent_id")
	if agentID == "" {
		agentID = c.GetString("agentID")
	}
	var row struct {
		OrgID           uuid.UUID  `db:"org_id"`
		AgentID         uuid.UUID  `db:"agent_id"`
		Violations      int        `db:"violations"`
		LastViolationAt *time.Time `db:"last_violation_at"`
		MTTVSeconds     *int       `db:"mttv_seconds"`
		Reliability     int        `db:"reliability_score"`
	}
	if err := database.DB.Get(&row, `SELECT org_id, agent_id, violations, last_violation_at, mttv_seconds, reliability_score FROM agent_reputation WHERE org_id=$1 AND agent_id=$2`, orgID, agentID); err != nil {
		c.JSON(http.StatusOK, gin.H{"org_id": orgID, "agent_id": agentID, "violations": 0, "reliability": 100})
		return
	}
	c.JSON(http.StatusOK, gin.H{"org_id": row.OrgID, "agent_id": row.AgentID, "violations": row.Violations, "reliability": row.Reliability, "last_violation_at": row.LastViolationAt, "mttv_seconds": row.MTTVSeconds})
}
