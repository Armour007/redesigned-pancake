-- +goose Up
CREATE TABLE IF NOT EXISTS runtime_approvals (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id uuid,
  trace_id text NOT NULL,
  status text NOT NULL DEFAULT 'pending', -- pending|approved|denied
  decided_by text,
  decided_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_runtime_approvals_trace ON runtime_approvals(trace_id);
CREATE INDEX IF NOT EXISTS idx_runtime_approvals_status ON runtime_approvals(status);

-- +goose Down
DROP TABLE IF EXISTS runtime_approvals;
