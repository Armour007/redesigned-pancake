package api

import (
	"net/http"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/Armour007/aura-backend/internal/audit"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OrganizationResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetMyOrganizations returns organizations the current user belongs to.
func GetMyOrganizations(c *gin.Context) {
	userID, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	type row struct{ ID, Name string }
	rows := []row{}
	err := database.DB.Select(&rows, `
        SELECT o.id, o.name
        FROM organizations o
        JOIN organization_members m ON m.organization_id = o.id
        WHERE m.user_id = $1
        ORDER BY o.created_at ASC`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch organizations"})
		return
	}

	resp := make([]OrganizationResponse, 0, len(rows))
	for _, r := range rows {
		resp = append(resp, OrganizationResponse{ID: r.ID, Name: r.Name})
	}
	c.JSON(http.StatusOK, resp)
}

// GetOrganizationByID returns a single organization's basic info by ID (requires membership)
func GetOrganizationByID(c *gin.Context) {
	orgID := c.Param("orgId")
	var row struct{ ID, Name string }
	err := database.DB.Get(&row, `SELECT id, name FROM organizations WHERE id=$1`, orgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Organization not found"})
		return
	}
	c.JSON(http.StatusOK, OrganizationResponse{ID: row.ID, Name: row.Name})
}

type UpdateOrganizationRequest struct {
	Name string `json:"name"`
}

// UpdateOrganization updates mutable org settings (currently only name). Admin only.
func UpdateOrganization(c *gin.Context) {
	orgID := c.Param("orgId")
	var req UpdateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid name"})
		return
	}
	_, err := database.DB.Exec(`UPDATE organizations SET name=$1, updated_at=NOW() WHERE id=$2`, req.Name, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update organization"})
		return
	}
	GetOrganizationByID(c)
}

// UpdateOrganizationSettings toggles org-level flags like api_keys_disabled. Admin only.
type UpdateOrganizationSettingsRequest struct {
	ApiKeysDisabled *bool `json:"api_keys_disabled"`
}

// PUT /organizations/:orgId/settings
func UpdateOrganizationSettings(c *gin.Context) {
	orgID := c.Param("orgId")
	var req UpdateOrganizationSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if req.ApiKeysDisabled == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "api_keys_disabled required"})
		return
	}
	if _, err := database.DB.Exec(`UPDATE organizations SET api_keys_disabled=$1, updated_at=NOW() WHERE id=$2`, *req.ApiKeysDisabled, orgID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update settings"})
		return
	}
	// audit
	var actor *uuid.UUID
	if uid := c.GetString("userID"); uid != "" {
		u := uuid.MustParse(uid)
		actor = &u
	}
	_ = audit.Append(c.Request.Context(), uuid.MustParse(orgID), "org_settings_update", map[string]any{"api_keys_disabled": *req.ApiKeysDisabled}, actor, nil)
	c.JSON(http.StatusOK, gin.H{"id": orgID, "api_keys_disabled": *req.ApiKeysDisabled})
}
