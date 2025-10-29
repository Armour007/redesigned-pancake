package api

import (
	"net/http"

	db "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GET /organizations/:orgId/policies/:policyId/versions
func ListPolicyVersions(c *gin.Context) {
	pid, err := uuid.Parse(c.Param("policyId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad policy id"})
		return
	}
	var rows []db.PolicyVersion
	if err := db.DB.Select(&rows, `SELECT id, policy_id, version, body, compiled_blob, checksum, status, created_by_user_id, created_at, approved_by_user_id, approved_at, activated_at FROM policy_versions WHERE policy_id=$1 ORDER BY version DESC`, pid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rows)
}
