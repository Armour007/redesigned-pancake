package api

import (
	"net/http"

	db "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ListPolicyVersionsV2 lists versions for a policy, scoped to the caller's org from auth context.
// GET /v2/policies/:policyId/versions
func ListPolicyVersionsV2(c *gin.Context) {
	orgID := c.GetString("orgID")
	pid, err := uuid.Parse(c.Param("policyId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad policy id"})
		return
	}
	// Restrict results to policies that belong to the caller's org
	var rows []struct {
		ID          uuid.UUID `db:"id" json:"id"`
		PolicyID    uuid.UUID `db:"policy_id" json:"policy_id"`
		Version     int       `db:"version" json:"version"`
		Status      string    `db:"status" json:"status"`
		CreatedAt   string    `db:"created_at" json:"created_at"`
		ApprovedAt  *string   `db:"approved_at" json:"approved_at"`
		ActivatedAt *string   `db:"activated_at" json:"activated_at"`
	}
	if err := db.DB.Select(&rows, `
        SELECT pv.id, pv.policy_id, pv.version, pv.status, pv.created_at::text, pv.approved_at::text, pv.activated_at::text
        FROM policy_versions pv
        JOIN policies p ON p.id = pv.policy_id
        WHERE pv.policy_id=$1 AND p.org_id=$2
        ORDER BY pv.version DESC
    `, pid, orgID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"policy_id": pid, "versions": rows})
}
