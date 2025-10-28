package api

import (
	"net/http"
	"time"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CreateWebhookRequest struct {
	URL    string `json:"url"`
	Secret string `json:"secret"`
}

func CreateWebhookEndpoint(c *gin.Context) {
	orgIDStr := c.Param("orgId")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}
	var req CreateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.URL == "" || req.Secret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	wh := database.WebhookEndpoint{
		ID:             uuid.New(),
		OrganizationID: orgID,
		URL:            req.URL,
		Secret:         req.Secret,
		IsActive:       true,
		CreatedAt:      time.Now(),
	}
	_, err = database.DB.NamedExec(`INSERT INTO webhook_endpoints (id, organization_id, url, secret, is_active, created_at)
        VALUES (:id, :organization_id, :url, :secret, :is_active, :created_at)`, wh)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create webhook endpoint"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": wh.ID, "url": wh.URL, "is_active": wh.IsActive, "created_at": wh.CreatedAt})
}

func ListWebhookEndpoints(c *gin.Context) {
	orgIDStr := c.Param("orgId")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}
	var items []struct {
		ID        uuid.UUID `db:"id" json:"id"`
		URL       string    `db:"url" json:"url"`
		IsActive  bool      `db:"is_active" json:"is_active"`
		CreatedAt time.Time `db:"created_at" json:"created_at"`
	}
	err = database.DB.Select(&items, `SELECT id, url, is_active, created_at FROM webhook_endpoints WHERE organization_id=$1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list webhook endpoints"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func DeleteWebhookEndpoint(c *gin.Context) {
	orgIDStr := c.Param("orgId")
	whIDStr := c.Param("webhookId")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}
	whID, err := uuid.Parse(whIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook ID"})
		return
	}
	// Constrain deletion by org to prevent cross-tenant deletes
	res, err := database.DB.Exec(`DELETE FROM webhook_endpoints WHERE id=$1 AND organization_id=$2`, whID, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete webhook endpoint"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Webhook endpoint not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
