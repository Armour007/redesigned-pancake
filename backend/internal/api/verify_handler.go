package api

import (
	"encoding/json" // Make sure this is imported
	"log"
	"net" // Make sure this is imported
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/Armour007/aura-backend/internal/engine"
)

// HandleVerifyRequest handles the core permission verification requests
func HandleVerifyRequest(c *gin.Context) {
	var req VerifyRequest

	// --- Placeholder Section ---
	// TODO: Implement API Key Authentication Middleware...
	placeholderAPIKeyPrefix := "tempkey"

	// Replace uuid.Nil with a REAL org ID from your database for testing
	// Get this ID from a successful user registration response in Postman
	orgIdStringForParsing := "2bc40ca7-7830-4e3a-8f17-daf017247bb9" // Use a temporary variable name
	placeholderOrgID, err := uuid.Parse(orgIdStringForParsing)      // Use := to declare err here
	if err != nil {
		log.Printf("FATAL: Could not parse placeholder Org ID '%s': %v", orgIdStringForParsing, err) // Log the string that failed
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Server configuration error (placeholder OrgID)"})
		return
	}
	// --- End Placeholder Section ---

	// Bind JSON request body and validate
	if err := c.ShouldBindJSON(&req); err != nil { // Use := to declare err in this block
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		go logEvent(c, req, "DENIED", "Invalid request body", placeholderAPIKeyPrefix, placeholderOrgID)
		return
	}

	// Basic validation: Is the provided context valid JSON?
	if !json.Valid(req.RequestContext) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'request_context' must contain valid JSON"})
		go logEvent(c, req, "DENIED", "Invalid request context format", placeholderAPIKeyPrefix, placeholderOrgID)
		return
	}

	// Call the core evaluation engine
	allowed, reason := engine.Evaluate(req.AgentID, req.RequestContext)

	// Determine decision string
	decision := "DENIED"
	if allowed {
		decision = "ALLOWED"
	}

	// Log the event asynchronously
	go logEvent(c, req, decision, reason, placeholderAPIKeyPrefix, placeholderOrgID)

	// Respond with the decision
	c.JSON(http.StatusOK, VerifyResponse{
		Decision: decision,
		Reason:   reason,
	})
}

// logEvent helper function (make sure net and encoding/json are imported)
func logEvent(c *gin.Context, req VerifyRequest, decision string, reason string, apiKeyPrefix string, orgID uuid.UUID) {
	clientIP := c.ClientIP()
	ipAddr := net.ParseIP(clientIP)

	// Use pointers for potentially null fields if your struct uses them
	// If ApiKeyPrefixUsed, DecisionReason, ClientIPAddress are *string/*net.IP
	var reasonPtr *string
	if reason != "" {
		reasonPtr = &reason
	}
	var apiKeyPrefixPtr *string
	if apiKeyPrefix != "" {
		apiKeyPrefixPtr = &apiKeyPrefix
	}
	var ipAddrPtr *net.IP
	if ipAddr != nil {
		ipAddrPtr = &ipAddr
	}

	event := database.EventLog{
		OrganizationID:   orgID,
		AgentID:          &req.AgentID, // Assuming AgentID in EventLog is *uuid.UUID
		Timestamp:        time.Now(),
		EventType:        "VERIFICATION",  // Make sure this field exists in your struct/DB
		ApiKeyPrefixUsed: apiKeyPrefixPtr, // Pass pointer
		Decision:         decision,
		RequestDetails:   req.RequestContext,
		DecisionReason:   reasonPtr, // Pass pointer
		ClientIPAddress:  ipAddrPtr, // Pass pointer
	}

	query := `INSERT INTO event_logs (organization_id, agent_id, timestamp, event_type, api_key_prefix_used, decision, request_details, decision_reason, client_ip_address)
	          VALUES (:organization_id, :agent_id, :timestamp, :event_type, :api_key_prefix_used, :decision, :request_details, :decision_reason, :client_ip_address)`

	_, err := database.DB.NamedExec(query, event) // Use = here as err exists
	if err != nil {
		log.Printf("ERROR logging event for agent %s: %v", req.AgentID, err)
	}
}
