-- +goose Up
CREATE TABLE "event_logs" (
  "id" bigserial PRIMARY KEY, -- Use bigserial for high-volume, ordered events
  "organization_id" uuid NOT NULL,
  "agent_id" uuid, -- Can be NULL if the event is not agent-specific (e.g., login attempt)
  "timestamp" timestamptz NOT NULL DEFAULT now(),
  "event_type" varchar(100) NOT NULL, -- e.g., 'VERIFICATION_REQUEST', 'RULE_CREATED', 'API_KEY_REVOKED'
  "api_key_prefix_used" varchar(8),
  "decision" varchar(50), -- e.g., 'ALLOWED', 'DENIED_POLICY', 'DENIED_INVALID_KEY', 'SUCCESS', 'FAILURE'
  "request_details" jsonb, -- The full request payload evaluated (if applicable)
  "decision_reason" text,
  "client_ip_address" inet,
  "user_id" uuid, -- The user who initiated the action (if applicable, e.g., changing a rule)
  -- Foreign key relationships
  FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE CASCADE,
  FOREIGN KEY ("agent_id") REFERENCES "agents" ("id") ON DELETE SET NULL,
  FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE SET NULL
);
-- Add indexes for common querying patterns
CREATE INDEX idx_event_logs_org_agent ON event_logs(organization_id, agent_id, timestamp DESC);
CREATE INDEX idx_event_logs_event_type ON event_logs(event_type, timestamp DESC);

-- +goose Down
DROP TABLE "event_logs";

