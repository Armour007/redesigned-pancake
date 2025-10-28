package api

import (
	"fmt"
	"net/http"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ensureAgentInOrg verifies the agent belongs to the given organization.
// On mismatch, it writes a 403 and returns false.
func ensureAgentInOrg(c *gin.Context, agentID uuid.UUID, orgID uuid.UUID) bool {
	var n int
	if err := database.DB.Get(&n, `SELECT COUNT(1) FROM agents WHERE id=$1 AND organization_id=$2`, agentID, orgID); err != nil || n == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Agent not in organization"})
		return false
	}
	return true
}

// ensureOrgOwns checks if a record identified by idColumn=id in table belongs to orgID.
// Use only with constant table/column names.
func ensureOrgOwns(c *gin.Context, table, idColumn string, id uuid.UUID, orgID uuid.UUID) bool {
	query := fmt.Sprintf("SELECT COUNT(1) FROM %s WHERE %s=$1 AND organization_id=$2", table, idColumn)
	var n int
	if err := database.DB.Get(&n, query, id, orgID); err != nil || n == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Resource not in organization"})
		return false
	}
	return true
}
