-- +goose Up
CREATE TABLE IF NOT EXISTS audit_ledger (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  seq bigserial NOT NULL,
  event_time timestamptz NOT NULL DEFAULT now(),
  actor_user_id uuid,
  actor_agent_id uuid,
  event_type text NOT NULL,
  payload jsonb NOT NULL,
  prev_hash text,
  this_hash text NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_audit_ledger_org_seq ON audit_ledger(org_id, seq);
CREATE INDEX IF NOT EXISTS idx_audit_ledger_org_time ON audit_ledger(org_id, event_time DESC);

-- +goose Down
DROP TABLE IF EXISTS audit_ledger;
