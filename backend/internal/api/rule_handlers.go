package api

import (
	"encoding/json" // Import json package
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	// Adjust import paths
	database "github.com/Armour007/aura-backend/internal"
)

// AddPermissionRule handles requests to add a new rule to an agent
func AddPermissionRule(c *gin.Context) {
	var req AddRuleRequest

	_ = c.Param("orgId") // We might use this later for auth
	agentIdStr := c.Param("agentId")

	// Validate request body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}
	// Basic validation: Is the provided rule valid JSON?
	if !json.Valid(req.Rule) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'rule' must contain valid JSON"})
		return
	}

	// Validate agentId from URL parameter
	agentId, err := uuid.Parse(agentIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID format"})
		return
	}

	// Get user ID from context (for potential authorization later)
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}
	// TODO: Add authorization checks:
	// 1. Does the user belong to orgId?
	// 2. Does the agent belong to orgId?

	// Create Permission struct for database insertion
	newPermission := database.Permission{
		ID:        uuid.New(),
		AgentID:   agentId,
		Rule:      req.Rule, // Store the raw JSON message
		IsActive:  true,     // Default to active
		CreatedAt: time.Now(),
	}

	// Insert into database
	query := `INSERT INTO permissions (id, agent_id, rule, is_active, created_at)
	          VALUES (:id, :agent_id, :rule, :is_active, :created_at)`
	_, err = database.DB.NamedExec(query, newPermission)
	if err != nil {
		// Basic error handling - could fail if agentId doesn't exist (foreign key)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add permission rule: " + err.Error()})
		return
	}

	// Respond with the created permission details
	response := PermissionResponse{
		ID:        newPermission.ID,
		AgentID:   newPermission.AgentID,
		Rule:      newPermission.Rule,
		IsActive:  newPermission.IsActive,
		CreatedAt: newPermission.CreatedAt,
	}
	c.JSON(http.StatusCreated, response)

	// Audit log (fire and forget)
	go func() {
		ua := c.Request.UserAgent()
		path := c.FullPath()
		rid := c.GetString("requestID")
		ridPtr, uaPtr, pathPtr := &rid, &ua, &path
		status := 201
		statusPtr := &status
		details := map[string]any{"agent_id": agentId.String()}
		event := database.EventLog{
			OrganizationID: func() uuid.UUID { org, _ := uuid.Parse(c.Param("orgId")); return org }(),
			AgentID:        &agentId,
			Timestamp:      time.Now(),
			EventType:      "RULE_CREATED",
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

// GetPermissionRules handles requests to list rules for an agent
// GetPermissionRules handles requests to list rules for an agent
func GetPermissionRules(c *gin.Context) {

	_ = c.Param("orgId") // For auth later
	agentIdStr := c.Param("agentId")

	// Validate agentId
	agentId, err := uuid.Parse(agentIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID format"})
		return
	}

	// Get user ID from context (for potential authorization later)
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}
	// TODO: Add authorization checks

	// Query database for permissions associated with this agent
	var permissions []database.Permission // Slice to hold multiple permissions
	query := `SELECT id, agent_id, rule, is_active, created_at
	          FROM permissions WHERE agent_id = $1 ORDER BY created_at DESC`

	err = database.DB.Select(&permissions, query, agentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve permission rules: " + err.Error()})
		return
	}

	// Convert database models to API response models
	responsePermissions := make([]PermissionResponse, 0, len(permissions))
	for _, perm := range permissions {
		responsePermissions = append(responsePermissions, PermissionResponse{
			ID:        perm.ID,
			AgentID:   perm.AgentID,
			Rule:      perm.Rule,
			IsActive:  perm.IsActive,
			CreatedAt: perm.CreatedAt,
		})
	}

	// Respond with the list of permissions
	c.JSON(http.StatusOK, responsePermissions)
}

// NOTE: You'll likely need a handler for deleting a rule too.

// DeletePermissionRule handles requests to delete a specific rule
// DeletePermissionRule handles requests to delete a specific rule
func DeletePermissionRule(c *gin.Context) {
	orgIdStr := c.Param("orgId") // For auth later
	agentIdStr := c.Param("agentId")
	ruleIdStr := c.Param("ruleId")

	// Validate IDs
	_, err := uuid.Parse(orgIdStr) // Parse orgId for potential auth check
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
		return
	}
	agentId, err := uuid.Parse(agentIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID format"})
		return
	}
	ruleId, err := uuid.Parse(ruleIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rule ID format"})
		return
	}

	// Get user ID from context
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}
	// TODO: Add authorization checks (user in org? agent in org? rule belongs to agent?)

	// Delete from database
	// We add a check for agent_id as well, so you can only delete rules belonging to the specified agent.
	query := `DELETE FROM permissions WHERE id = $1 AND agent_id = $2`
	result, err := database.DB.Exec(query, ruleId, agentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete permission rule: " + err.Error()})
		return
	}

	// Check if any row was actually deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check rows affected: " + err.Error()})
		return
	}
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found or does not belong to the specified agent"})
		return
	}

	// Respond with success (No Content)
	c.Status(http.StatusNoContent)

	// Audit log (fire and forget)
	go func() {
		ua := c.Request.UserAgent()
		path := c.FullPath()
		rid := c.GetString("requestID")
		ridPtr, uaPtr, pathPtr := &rid, &ua, &path
		status := 204
		statusPtr := &status
		details := map[string]any{"agent_id": agentId.String(), "rule_id": ruleId.String()}
		event := database.EventLog{
			OrganizationID: func() uuid.UUID { org, _ := uuid.Parse(c.Param("orgId")); return org }(),
			AgentID:        &agentId,
			Timestamp:      time.Now(),
			EventType:      "RULE_DELETED",
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
