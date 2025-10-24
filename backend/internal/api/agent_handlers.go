package api

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	// Adjust import paths
	database "github.com/Armour007/aura-backend/internal"
	//"github.com/Armour007/aura-backend/internal/utils" // Might need later
)

// CreateAgent handles requests to create a new agent
func CreateAgent(c *gin.Context) {
	var req CreateAgentRequest
	orgIdStr := c.Param("orgId") // Get organization ID from URL

	// Validate request body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Validate orgId from URL parameter
	orgId, err := uuid.Parse(orgIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
		return
	}

	// Get user ID from the middleware context (set during authentication)
	userIDStr, exists := c.Get("userID")
	if !exists {
		// This should technically not happen if middleware is applied correctly
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}
	userID, err := uuid.Parse(userIDStr.(string)) // Convert interface{} to string, then parse
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format in context"})
		return
	}

	// TODO: Add authorization check - does this user belong to this org?
	// For MVP, we might skip this complex check, but it's crucial for V1.0

	// Create Agent struct for database insertion
	newAgent := database.Agent{
		ID:              uuid.New(),
		OrganizationID:  orgId,
		Name:            req.Name,
		Description:     req.Description, // Pass pointer directly
		CreatedByUserID: userID,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Insert into database
	query := `INSERT INTO agents (id, organization_id, name, description, created_by_user_id, created_at, updated_at)
	          VALUES (:id, :organization_id, :name, :description, :created_by_user_id, :created_at, :updated_at)`
	_, err = database.DB.NamedExec(query, newAgent)
	if err != nil {
		// Basic error handling
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create agent: " + err.Error()})
		return
	}

	// Respond with the created agent details (using AgentResponse struct)
	response := AgentResponse{
		ID:             newAgent.ID,
		OrganizationID: newAgent.OrganizationID,
		Name:           newAgent.Name,
		Description:    newAgent.Description,
		CreatedAt:      newAgent.CreatedAt,
	}
	c.JSON(http.StatusCreated, response)
}

// GetAgents (Leave empty for now)
// GetAgents handles requests to list agents for an organization
func GetAgents(c *gin.Context) {
	orgIdStr := c.Param("orgId")

	// Validate orgId
	orgId, err := uuid.Parse(orgIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
		return
	}

	// Get user ID from context (for potential authorization later)
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}
	// TODO: Add authorization check - does this user belong to this org?

	// Query database for agents belonging to this organization
	var agents []database.Agent // Slice to hold multiple agents
	query := `SELECT id, organization_id, name, description, created_by_user_id, created_at, updated_at
	          FROM agents WHERE organization_id = $1 ORDER BY created_at DESC` // Use $1 for positional arg with sqlx.Select

	err = database.DB.Select(&agents, query, orgId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve agents: " + err.Error()})
		return
	}

	// Convert database models to API response models
	responseAgents := make([]AgentResponse, 0, len(agents)) // Efficiently create slice
	for _, agent := range agents {
		responseAgents = append(responseAgents, AgentResponse{
			ID:             agent.ID,
			OrganizationID: agent.OrganizationID,
			Name:           agent.Name,
			Description:    agent.Description,
			CreatedAt:      agent.CreatedAt,
		})
	}

	// Respond with the list of agents
	// If no agents are found, it will correctly return an empty list []
	c.JSON(http.StatusOK, responseAgents)
}

// UpdateAgent handles requests to update an existing agent
// UpdateAgent handles requests to update an existing agent
func UpdateAgent(c *gin.Context) {
	var req UpdateAgentRequest
	orgIdStr := c.Param("orgId")
	agentIdStr := c.Param("agentId")

	// Validate request body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Validate IDs
	orgId, err := uuid.Parse(orgIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
		return
	}
	agentId, err := uuid.Parse(agentIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID format"})
		return
	}

	// Get user ID from context
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}
	// TODO: Add authorization checks (user in org? agent in org?)

	// Build the UPDATE query dynamically based on provided fields
	// This is a bit more complex to handle optional fields correctly
	query := "UPDATE agents SET updated_at = :updated_at"
	params := map[string]interface{}{
		"id":         agentId,
		"org_id":     orgId, // Use orgId in WHERE clause for security
		"updated_at": time.Now(),
	}

	if req.Name != nil {
		query += ", name = :name"
		params["name"] = *req.Name
	}
	if req.Description != nil {
		query += ", description = :description"
		params["description"] = *req.Description
	} else if c.Request.Method == "PUT" && c.ContentType() == "application/json" {
		// Handle explicit null setting for description if needed (optional)
		// Check if 'description' key exists and is null in the raw JSON
	}

	query += " WHERE id = :id AND organization_id = :org_id" // Ensure user can only update agents in their org

	result, err := database.DB.NamedExec(query, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update agent: " + err.Error()})
		return
	}

	// Check if any row was actually updated
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check rows affected: " + err.Error()})
		return
	}
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found or not part of specified organization"})
		return
	}

	// Fetch the updated agent to return it (optional but good practice)
	var updatedAgent database.Agent
	getErr := database.DB.Get(&updatedAgent, "SELECT * FROM agents WHERE id = $1", agentId)
	if getErr != nil {
		// Log error but maybe still return success if update worked
		log.Printf("Error fetching updated agent %s: %v", agentId, getErr)
		c.JSON(http.StatusOK, gin.H{"message": "Agent updated successfully, but failed to fetch latest data"})
		return
	}

	response := AgentResponse{
		ID:             updatedAgent.ID,
		OrganizationID: updatedAgent.OrganizationID,
		Name:           updatedAgent.Name,
		Description:    updatedAgent.Description,
		CreatedAt:      updatedAgent.CreatedAt, // Note: returning CreatedAt, not UpdatedAt
	}

	c.JSON(http.StatusOK, response)
}

// DeleteAgent handles requests to delete an agent
// DeleteAgent handles requests to delete an agent
func DeleteAgent(c *gin.Context) {
	orgIdStr := c.Param("orgId")
	agentIdStr := c.Param("agentId")

	// Validate IDs
	orgId, err := uuid.Parse(orgIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
		return
	}
	agentId, err := uuid.Parse(agentIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID format"})
		return
	}

	// Get user ID from context
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}
	// TODO: Add authorization checks (user in org? agent in org?)

	// Delete from database
	query := `DELETE FROM agents WHERE id = $1 AND organization_id = $2`
	result, err := database.DB.Exec(query, agentId, orgId) // Use Exec for simple delete
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete agent: " + err.Error()})
		return
	}

	// Check if any row was actually deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check rows affected: " + err.Error()})
		return
	}
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found or not part of specified organization"})
		return
	}

	// Respond with success (No Content)
	c.Status(http.StatusNoContent)
}
