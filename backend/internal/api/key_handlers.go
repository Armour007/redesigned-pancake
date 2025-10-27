package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// CreateAPIKey handles requests to generate a new API key
func CreateAPIKey(c *gin.Context) {
	orgIdStr := c.Param("orgId")
	// Validate orgId
	orgID, err := uuid.Parse(orgIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
		return
	}

	// Bind request
	var req NewAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get user ID from context
	userIDStr, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID, _ := uuid.Parse(userIDStr.(string))

	// Generate secret key
	secretRaw, err := generateSecret()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate API key"})
		return
	}
	secret := "aura_sk_" + secretRaw
	keyPrefix := secretRaw[:8]

	// Hash secret
	hashed, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to secure API key"})
		return
	}

	// Build record
	key := database.APIKey{
		ID:              uuid.New(),
		OrganizationID:  orgID,
		Name:            req.Name,
		KeyPrefix:       keyPrefix,
		HashedKey:       string(hashed),
		CreatedByUserID: userID,
		ExpiresAt:       req.ExpiresAt,
		CreatedAt:       time.Now(),
	}

	// Insert
	_, err = database.DB.NamedExec(`INSERT INTO api_keys (id, organization_id, name, key_prefix, hashed_key, created_by_user_id, last_used_at, expires_at, created_at)
		VALUES (:id, :organization_id, :name, :key_prefix, :hashed_key, :created_by_user_id, :last_used_at, :expires_at, :created_at)`, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist API key"})
		return
	}

	// Build a convenience Quick Start link to help first-time users.
	// Prefer AURA_FRONTEND_BASE_URL, fallback to PUBLIC_SITE_URL, else localhost dev URL.
	frontendBase := strings.TrimRight(func() string {
		if v := os.Getenv("AURA_FRONTEND_BASE_URL"); v != "" {
			return v
		}
		if v := os.Getenv("PUBLIC_SITE_URL"); v != "" {
			return v
		}
		return "http://localhost:5173"
	}(), "/")
	quickstartURL := frontendBase + "/quickstart?key_prefix=" + keyPrefix

	// Respond with secret ONCE plus helper URL
	c.JSON(http.StatusCreated, NewAPIKeyResponse{
		ID:            key.ID,
		Name:          key.Name,
		KeyPrefix:     key.KeyPrefix,
		SecretKey:     secret,
		CreatedAt:     key.CreatedAt,
		ExpiresAt:     key.ExpiresAt,
		QuickStartURL: quickstartURL,
	})

	// Audit log (fire and forget)
	go func() {
		ua := c.Request.UserAgent()
		path := c.FullPath()
		rid := c.GetString("requestID")
		orgID := orgID
		ridPtr, uaPtr, pathPtr := &rid, &ua, &path
		status := 201
		statusPtr := &status
		// store lightweight details
		details := map[string]any{"name": req.Name, "expires_at": req.ExpiresAt}
		event := database.EventLog{
			OrganizationID: orgID,
			AgentID:        nil,
			Timestamp:      time.Now(),
			EventType:      "API_KEY_CREATED",
			Decision:       "SUCCESS",
			RequestDetails: toJSON(details),
			DecisionReason: nil,
			ClientIPAddress: func() *net.IP {
				ip := net.ParseIP(c.ClientIP())
				if ip != nil {
					return &ip
				}
				return nil
			}(),
			RequestID:  ridPtr,
			UserAgent:  uaPtr,
			Path:       pathPtr,
			StatusCode: statusPtr,
		}
		_, _ = database.DB.NamedExec(`INSERT INTO event_logs (organization_id, agent_id, timestamp, event_type, decision, request_details, client_ip_address, request_id, user_agent, path, status_code)
			VALUES (:organization_id, :agent_id, :timestamp, :event_type, :decision, :request_details, :client_ip_address, :request_id, :user_agent, :path, :status_code)`, event)
	}()
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
			  FROM api_keys WHERE organization_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC`

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
	orgIdStr := c.Param("orgId")
	keyIdStr := c.Param("keyId")

	orgID, err := uuid.Parse(orgIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}
	keyID, err := uuid.Parse(keyIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid key ID"})
		return
	}

	// Soft-revoke by setting revoked_at
	result, err := database.DB.Exec(`UPDATE api_keys SET revoked_at=$1 WHERE id=$2 AND organization_id=$3 AND revoked_at IS NULL`, time.Now(), keyID, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke API key"})
		return
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found or already revoked"})
		return
	}
	c.Status(http.StatusNoContent)

	// Audit log (fire and forget)
	go func() {
		ua := c.Request.UserAgent()
		path := c.FullPath()
		rid := c.GetString("requestID")
		ridPtr, uaPtr, pathPtr := &rid, &ua, &path
		status := 204
		statusPtr := &status
		details := map[string]any{"key_id": keyID.String()}
		event := database.EventLog{
			OrganizationID: orgID,
			AgentID:        nil,
			Timestamp:      time.Now(),
			EventType:      "API_KEY_REVOKED",
			Decision:       "SUCCESS",
			RequestDetails: toJSON(details),
			DecisionReason: nil,
			ClientIPAddress: func() *net.IP {
				ip := net.ParseIP(c.ClientIP())
				if ip != nil {
					return &ip
				}
				return nil
			}(),
			RequestID:  ridPtr,
			UserAgent:  uaPtr,
			Path:       pathPtr,
			StatusCode: statusPtr,
		}
		_, _ = database.DB.NamedExec(`INSERT INTO event_logs (organization_id, agent_id, timestamp, event_type, decision, request_details, client_ip_address, request_id, user_agent, path, status_code)
			VALUES (:organization_id, :agent_id, :timestamp, :event_type, :decision, :request_details, :client_ip_address, :request_id, :user_agent, :path, :status_code)`, event)
	}()
}

// toJSON marshals a map into json.RawMessage without failing the outer handler
func toJSON(v map[string]any) (out []byte) {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}

// generateSecret returns a 64-hex-character random string
func generateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
