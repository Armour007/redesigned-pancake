package api

import (
	"bytes"
	"context"
	"encoding/json" // Make sure this is imported
	"fmt"
	"log"
	"net" // Make sure this is imported
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	database "github.com/Armour007/aura-backend/internal"
	"github.com/Armour007/aura-backend/internal/engine"
	"github.com/Armour007/aura-backend/internal/utils"
)

// HandleVerifyRequest handles the core permission verification requests
func HandleVerifyRequest(c *gin.Context) {
	ctx, span := otel.Tracer("aura-backend").Start(c.Request.Context(), "verify")
	defer span.End()
	var req VerifyRequest

	// Extract API key context from middleware
	orgIDStr, ok := c.Get("orgID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization context missing"})
		return
	}
	orgID, err := uuid.Parse(orgIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid organization context"})
		return
	}
	span.SetAttributes(attribute.String("org_id", orgID.String()))
	apiKeyPrefix, _ := c.Get("apiKeyPrefix")

	// Bind JSON request body and validate
	if err := c.ShouldBindJSON(&req); err != nil { // Use := to declare err in this block
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		go logEvent(c, req, "DENIED", "Invalid request body", asString(apiKeyPrefix), orgID)
		span.SetStatus(codes.Error, "bind json")
		return
	}
	span.SetAttributes(attribute.String("agent_id", req.AgentID.String()))

	// Basic validation: Is the provided context valid JSON?
	if !json.Valid(req.RequestContext) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Field 'request_context' must contain valid JSON"})
		go logEvent(c, req, "DENIED", "Invalid request context format", asString(apiKeyPrefix), orgID)
		span.SetStatus(codes.Error, "invalid context json")
		return
	}

	// Ensure the agent belongs to this organization
	var count int
	{ // span for DB membership check
		dbctx, dbspan := otel.Tracer("aura-backend").Start(ctx, "db.agent_membership")
		defer dbspan.End()
		err = database.DB.GetContext(dbctx, &count, `SELECT COUNT(1) FROM agents WHERE id=$1 AND organization_id=$2`, req.AgentID, orgID)
	}
	if err != nil || count == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Agent not found in organization"})
		go logEvent(c, req, "DENIED", "Agent not in organization", asString(apiKeyPrefix), orgID)
		span.SetStatus(codes.Error, "agent not in org")
		return
	}

	// Call the core evaluation engine
	var allowed bool
	var reason string
	{ // span for evaluation
		_, evspan := otel.Tracer("aura-backend").Start(ctx, "engine.evaluate")
		allowed, reason = engine.Evaluate(req.AgentID, req.RequestContext)
		evspan.SetAttributes(attribute.String("reason", reason))
		evspan.End()
	}

	// Determine decision string
	decision := "DENIED"
	if allowed {
		decision = "ALLOWED"
	}

	// record metrics for decision outcome with org label
	RecordDecision(decision, orgID.String())
	RecordDecisionReason(reason, decision, orgID.String())
	span.SetAttributes(attribute.String("decision", decision), attribute.String("reason", reason))

	// Log the event asynchronously (offload heavy writes if configured)
	go func() {
		logEvent(c, req, decision, reason, asString(apiKeyPrefix), orgID)
		// After logging, emit webhook to any active endpoints for this org
		// Build a compact event payload similar to Stripe events
		payload := map[string]any{
			"type": "verification.decision",
			"data": map[string]any{
				"organization_id": orgID.String(),
				"agent_id":        req.AgentID.String(),
				"decision":        decision,
				"reason":          reason,
				"request_context": json.RawMessage(req.RequestContext),
				"timestamp":       time.Now().UTC().Format(time.RFC3339),
			},
		}
		if b, err := json.Marshal(payload); err == nil {
			dispatchWebhooks(orgID, "verification.decision", b)
		}
	}()

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
	// Try to include Request ID into the logged request_details for traceability
	requestID := c.GetString("requestID")
	userAgent := c.Request.UserAgent()
	path := c.FullPath()
	status := 200 // verify endpoint always responds 200 with decision payload
	var augmented json.RawMessage
	if len(req.RequestContext) > 0 {
		var ctxMap map[string]interface{}
		if err := json.Unmarshal(req.RequestContext, &ctxMap); err == nil {
			if requestID != "" {
				ctxMap["request_id"] = requestID
			}
			if data, err := json.Marshal(ctxMap); err == nil {
				augmented = json.RawMessage(data)
			}
		}
	}

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
	var reqIDPtr *string
	if requestID != "" {
		reqIDPtr = &requestID
	}
	var uaPtr *string
	if userAgent != "" {
		uaPtr = &userAgent
	}
	var pathPtr *string
	if path != "" {
		pathPtr = &path
	}
	var statusPtr *int
	statusPtr = &status

	event := database.EventLog{
		OrganizationID:   orgID,
		AgentID:          &req.AgentID, // Assuming AgentID in EventLog is *uuid.UUID
		Timestamp:        time.Now(),
		EventType:        "VERIFICATION",  // Make sure this field exists in your struct/DB
		ApiKeyPrefixUsed: apiKeyPrefixPtr, // Pass pointer
		Decision:         decision,
		RequestDetails: func() json.RawMessage {
			if len(augmented) > 0 {
				return augmented
			}
			return req.RequestContext
		}(),
		DecisionReason:  reasonPtr, // Pass pointer
		ClientIPAddress: ipAddrPtr, // Pass pointer
		RequestID:       reqIDPtr,
		UserAgent:       uaPtr,
		Path:            pathPtr,
		StatusCode:      statusPtr,
	}

	// Offload to queue if configured
	if mode := os.Getenv("AURA_EVENTLOG_OFFLOAD"); mode == "redis" && redisClient != nil {
		payload, _ := json.Marshal(event)
		_ = redisClient.XAdd(context.Background(), &redis.XAddArgs{Stream: "aura:event_logs", Values: map[string]any{"payload": string(payload)}}).Err()
		return
	}
	query := `INSERT INTO event_logs (organization_id, agent_id, timestamp, event_type, api_key_prefix_used, decision, request_details, decision_reason, client_ip_address, request_id, user_agent, path, status_code)
			  VALUES (:organization_id, :agent_id, :timestamp, :event_type, :api_key_prefix_used, :decision, :request_details, :decision_reason, :client_ip_address, :request_id, :user_agent, :path, :status_code)`

	_, err := database.DB.NamedExec(query, event) // Use = here as err exists
	if err != nil {
		log.Printf("ERROR logging event for agent %s: %v", req.AgentID, err)
	}
}

// asString safely converts a value to string
func asString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// dispatchWebhooks posts the given payload to all active endpoints for org,
// signing the payload with each endpoint's secret using header: AURA-Signature: t=timestamp,v1=hex
func dispatchWebhooks(orgID uuid.UUID, eventType string, payload []byte) {
	// lookup endpoints
	var endpoints []database.WebhookEndpoint
	if err := database.DB.Select(&endpoints, `SELECT id, organization_id, url, secret, is_active, created_at FROM webhook_endpoints WHERE organization_id=$1 AND is_active=true`, orgID); err != nil {
		log.Printf("webhook dispatch lookup failed: %v", err)
		return
	}
	if len(endpoints) == 0 {
		return
	}
	for _, ep := range endpoints {
		eventID := uuid.New().String()
		attempts := 3
		var lastStatus int
		for i := 0; i < attempts; i++ {
			ts := time.Now().Unix()
			sig := utils.ComputeWebhookSignature(ep.Secret, ts, payload)
			req, err := http.NewRequest("POST", ep.URL, bytes.NewReader(payload))
			if err != nil {
				break
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("AURA-Event", eventType)
			req.Header.Set("AURA-Webhook-ID", ep.ID.String())
			req.Header.Set("AURA-Event-ID", eventID)
			req.Header.Set("Idempotency-Key", eventID)
			req.Header.Set("AURA-Signature", fmt.Sprintf("t=%d,v1=%s", ts, sig))
			client := &http.Client{Timeout: 3 * time.Second}
			resp, err := client.Do(req)
			if err == nil {
				lastStatus = resp.StatusCode
				_ = resp.Body.Close()
			}
			if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
				break
			}
			// backoff
			time.Sleep(time.Duration(500*(1<<i)) * time.Millisecond)
		}
		// If final attempt failed or non-2xx, push to DLQ
		if lastStatus < 200 || lastStatus >= 300 {
			if redisClient != nil {
				_ = redisClient.XAdd(context.Background(), &redis.XAddArgs{Stream: "aura:webhooks:dlq", Values: map[string]any{
					"org_id":    orgID.String(),
					"endpoint":  ep.ID.String(),
					"url":       ep.URL,
					"event":     eventType,
					"payload":   string(payload),
					"attempts":  attempts,
					"last_code": lastStatus,
					"at":        time.Now().Unix(),
				}}).Err()
				RecordDLQInsert("webhooks", "delivery_failed")
				if x, err := redisClient.XLen(context.Background(), "aura:webhooks:dlq").Result(); err == nil {
					SetDLQDepth("webhooks", x)
				}
			}
		}
	}
}
