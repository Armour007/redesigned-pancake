package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreateAPIKey handles requests to generate a new API key
func CreateAPIKey(c *gin.Context) {
	orgId := c.Param("orgId")
	// TODO: Implement logic to create an API key
	c.JSON(http.StatusNotImplemented, gin.H{"message": "CreateAPIKey not implemented yet", "orgId": orgId})
}

// GetAPIKeys handles requests to list API keys for an organization
func GetAPIKeys(c *gin.Context) {
	orgId := c.Param("orgId")
	// TODO: Implement logic to get API keys (remember not to return the secret key itself!)
	c.JSON(http.StatusNotImplemented, gin.H{"message": "GetAPIKeys not implemented yet", "orgId": orgId})
}

// NOTE: You'll likely need a handler for deleting/revoking a key too.

// DeleteAPIKey handles requests to revoke an API key
func DeleteAPIKey(c *gin.Context) {
	orgId := c.Param("orgId")
	keyId := c.Param("keyId") // We'll need to add "/:keyId" route in main.go later
	// TODO: Implement logic to delete/revoke an API key
	c.JSON(http.StatusNotImplemented, gin.H{"message": "DeleteAPIKey not implemented yet", "orgId": orgId, "keyId": keyId})
}
