package api

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// RegisterRequest defines the expected JSON body for user registration
type RegisterRequest struct {
	FullName string `json:"full_name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"` // Enforce minimum password length
}
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}
type CreateAgentRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"` // Optional field
}

// AgentResponse defines the standard structure for returning agent data
// (excluding sensitive info)
type AgentResponse struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	Description    *string   `json:"description,omitempty"` // omitempty hides it if null/empty
	CreatedAt      time.Time `json:"created_at"`
}

// AddRuleRequest defines the expected JSON body for adding a rule
type AddRuleRequest struct {
	// We expect the user to send the rule as a nested JSON object
	Rule json.RawMessage `json:"rule" binding:"required"`
}

// PermissionResponse defines the structure for returning rule data
type PermissionResponse struct {
	ID        uuid.UUID       `json:"id"`
	AgentID   uuid.UUID       `json:"agent_id"`
	Rule      json.RawMessage `json:"rule"`
	IsActive  bool            `json:"is_active"`
	CreatedAt time.Time       `json:"created_at"`
}

// VerifyRequest defines the expected JSON body for the /v1/verify endpoint
type VerifyRequest struct {
	AgentID        uuid.UUID       `json:"agent_id" binding:"required"`
	RequestContext json.RawMessage `json:"request_context" binding:"required"` // The action payload to evaluate
}

// VerifyResponse defines the JSON response for the /v1/verify endpoint
type VerifyResponse struct {
	Decision string `json:"decision"`         // "ALLOWED" or "DENIED"
	Reason   string `json:"reason,omitempty"` // Optional explanation for denial
}
type UpdateAgentRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}
