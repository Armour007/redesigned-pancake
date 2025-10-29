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
	ID              uuid.UUID `db:"id"`
	Name            string    `db:"name"`
	OwnerID         uuid.UUID `db:"owner_id"`
	ApiKeysDisabled bool      `db:"api_keys_disabled"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
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
	RequestID        *string         `db:"request_id"`
	UserAgent        *string         `db:"user_agent"`
	Path             *string         `db:"path"`
	StatusCode       *int            `db:"status_code"`
}
type APIKey struct {
	ID              uuid.UUID  `db:"id"`
	OrganizationID  uuid.UUID  `db:"organization_id"`
	Name            string     `db:"name"`
	KeyPrefix       string     `db:"key_prefix"`
	HashedKey       string     `db:"hashed_key"` // Never return this via API
	CreatedByUserID uuid.UUID  `db:"created_by_user_id"`
	LastUsedAt      *time.Time `db:"last_used_at"` // Pointer for nullable
	ExpiresAt       *time.Time `db:"expires_at"`   // Pointer for nullable
	CreatedAt       time.Time  `db:"created_at"`
}

// WebhookEndpoint represents webhook subscription for an organization
type WebhookEndpoint struct {
	ID             uuid.UUID `db:"id"`
	OrganizationID uuid.UUID `db:"organization_id"`
	URL            string    `db:"url"`
	Secret         string    `db:"secret"`
	IsActive       bool      `db:"is_active"`
	CreatedAt      time.Time `db:"created_at"`
}

// Policy entities (DB-backed)
type Policy struct {
	ID         uuid.UUID  `db:"id"`
	OrgID      uuid.UUID  `db:"org_id"`
	Name       string     `db:"name"`
	EngineType string     `db:"engine_type"`
	CreatedBy  *uuid.UUID `db:"created_by_user_id"`
	CreatedAt  time.Time  `db:"created_at"`
}

type PolicyVersion struct {
	ID          uuid.UUID       `db:"id"`
	PolicyID    uuid.UUID       `db:"policy_id"`
	Version     int             `db:"version"`
	Body        json.RawMessage `db:"body"`
	Compiled    []byte          `db:"compiled_blob"`
	Checksum    *string         `db:"checksum"`
	Status      string          `db:"status"`
	CreatedBy   *uuid.UUID      `db:"created_by_user_id"`
	CreatedAt   time.Time       `db:"created_at"`
	ApprovedBy  *uuid.UUID      `db:"approved_by_user_id"`
	ApprovedAt  *time.Time      `db:"approved_at"`
	ActivatedAt *time.Time      `db:"activated_at"`
}

type PolicyAssignment struct {
	ID        uuid.UUID `db:"id"`
	PolicyID  uuid.UUID `db:"policy_id"`
	ScopeType string    `db:"scope_type"`
	ScopeID   string    `db:"scope_id"`
	CreatedAt time.Time `db:"created_at"`
}

type TrustTuple struct {
	ID          int64            `db:"id"`
	ObjectType  string           `db:"object_type"`
	ObjectID    string           `db:"object_id"`
	Relation    string           `db:"relation"`
	SubjectType string           `db:"subject_type"`
	SubjectID   string           `db:"subject_id"`
	CaveatJSON  *json.RawMessage `db:"caveat_json"`
	CreatedAt   time.Time        `db:"created_at"`
}

type DecisionTrace struct {
	ID            int64           `db:"id"`
	OrgID         uuid.UUID       `db:"org_id"`
	TraceID       string          `db:"trace_id"`
	PolicyID      *uuid.UUID      `db:"policy_id"`
	PolicyVersion *int            `db:"policy_version"`
	AgentID       *uuid.UUID      `db:"agent_id"`
	Allow         bool            `db:"allow"`
	Reason        *string         `db:"reason"`
	Trace         json.RawMessage `db:"trace"`
	CreatedAt     time.Time       `db:"created_at"`
}
