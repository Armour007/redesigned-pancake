package api

import (
	"net/http"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateAPIKey handles requests to generate a new API key
func CreateAPIKey(c *gin.Context) {
	orgId := c.Param("orgId")
	// TODO: Implement logic to create an API key
	c.JSON(http.StatusNotImplemented, gin.H{"message": "CreateAPIKey not implemented yet", "orgId": orgId})
}

// GetAPIKeys handles requests to list API keys for an organization
// GetAPIKeys handles requests to list API keys for an organization
func GetAPIKeys(c *gin.Context) {
	orgIdStr := c.Param("orgId")

	// Validate orgId
	orgId, err := uuid.Parse(orgIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
		return
	}

	// Get user ID from context
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}
	// TODO: Add authorization checks (user in org?)

	// Query database for API keys belonging to this organization
	var apiKeys []database.APIKey // Slice to hold multiple keys
	// Select only the fields needed for the response, exclude hashed_key
	query := `SELECT id, organization_id, name, key_prefix, created_by_user_id, created_at, last_used_at, expires_at
              FROM api_keys WHERE organization_id = $1 ORDER BY created_at DESC`

	err = database.DB.Select(&apiKeys, query, orgId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve API keys: " + err.Error()})
		return
	}

	// Convert database models to safe API response models
	responseKeys := make([]APIKeyInfoResponse, 0, len(apiKeys))
	for _, key := range apiKeys {
		responseKeys = append(responseKeys, APIKeyInfoResponse{
			ID:         key.ID,
			Name:       key.Name,
			KeyPrefix:  key.KeyPrefix,
			CreatedAt:  key.CreatedAt,
			LastUsedAt: key.LastUsedAt,
			ExpiresAt:  key.ExpiresAt,
		})
	}

	// Respond with the list of keys
	c.JSON(http.StatusOK, responseKeys)
}

// NOTE: You'll likely need a handler for deleting/revoking a key too.

// DeleteAPIKey handles requests to revoke an API key
func DeleteAPIKey(c *gin.Context) {
	orgId := c.Param("orgId")
	keyId := c.Param("keyId") // We'll need to add "/:keyId" route in main.go later
	// TODO: Implement logic to delete/revoke an API key
	c.JSON(http.StatusNotImplemented, gin.H{"message": "DeleteAPIKey not implemented yet", "orgId": orgId, "keyId": keyId})
}
