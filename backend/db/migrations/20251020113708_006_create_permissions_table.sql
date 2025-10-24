-- +goose Up
CREATE TABLE "permissions" (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "agent_id" uuid NOT NULL,
  -- Using JSONB for flexible rule storage
  "rule" jsonb NOT NULL,
  "description" text, -- Optional description for the rule
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  -- Foreign key relationship
  FOREIGN KEY ("agent_id") REFERENCES "agents" ("id") ON DELETE CASCADE
);
-- Add an index for faster lookups of rules for a specific agent
CREATE INDEX idx_permissions_agent_id ON permissions(agent_id);

-- +goose Down
DROP TABLE "permissions";

