package database

import (
	"encoding/json"
	"net"
	"time"

	"github.com/google/uuid"
)

// Agent represents the 'agents' table in the database
type Agent struct {
	ID              uuid.UUID `db:"id"`
	OrganizationID  uuid.UUID `db:"organization_id"` // Assuming we link agents to orgs
	Name            string    `db:"name"`
	Description     *string   `db:"description"` // Use *string for nullable fields
	CreatedByUserID uuid.UUID `db:"created_by_user_id"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

// You can add other database model structs here later (e.g., Organization, Permission)

// Organization represents the 'organizations' table
type Organization struct {
	ID        uuid.UUID `db:"id"`
	Name      string    `db:"name"`
	OwnerID   uuid.UUID `db:"owner_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// OrganizationMember represents the 'organization_members' table
type OrganizationMember struct {
	OrganizationID uuid.UUID `db:"organization_id"`
	UserID         uuid.UUID `db:"user_id"`
	Role           string    `db:"role"` // e.g., "owner", "admin", "member"
	JoinedAt       time.Time `db:"joined_at"`
}

// Permission represents the 'permissions' table
type Permission struct {
	ID        uuid.UUID       `db:"id"`
	AgentID   uuid.UUID       `db:"agent_id"`
	Rule      json.RawMessage `db:"rule"` // Use json.RawMessage to store raw JSON
	IsActive  bool            `db:"is_active"`
	CreatedAt time.Time       `db:"created_at"`
}

// EventLog represents the 'event_logs' table
type EventLog struct {
	ID               int64           `db:"id"`              // bigserial maps to int64
	OrganizationID   uuid.UUID       `db:"organization_id"` // Need to figure out how to get this efficiently
	AgentID          *uuid.UUID      `db:"agent_id"`        // Pointer for nullable
	Timestamp        time.Time       `db:"timestamp"`
	EventType        string          `db:"event_type"`
	ApiKeyPrefixUsed *string         `db:"api_key_prefix_used"` // Pointer for nullable
	Decision         string          `db:"decision"`
	RequestDetails   json.RawMessage `db:"request_details"`
	DecisionReason   *string         `db:"decision_reason"`   // Pointer for nullable
	ClientIPAddress  *net.IP         `db:"client_ip_address"` // Pointer for nullable
}
