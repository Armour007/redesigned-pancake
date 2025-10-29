-- +goose Up
CREATE TABLE IF NOT EXISTS decision_traces (
  id bigserial PRIMARY KEY,
  org_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  trace_id text NOT NULL,
  policy_id uuid,
  policy_version int,
  agent_id uuid,
  allow boolean NOT NULL,
  reason text,
  trace jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_decision_traces_trace ON decision_traces(trace_id);
CREATE INDEX IF NOT EXISTS idx_decision_traces_org_time ON decision_traces(org_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS decision_traces;
