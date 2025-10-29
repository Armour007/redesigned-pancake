CREATE TABLE IF NOT EXISTS agent_reputation (
  org_id uuid NOT NULL,
  agent_id uuid NOT NULL,
  violations integer NOT NULL DEFAULT 0,
  last_violation_at timestamptz,
  mttv_seconds integer,
  reliability_score integer NOT NULL DEFAULT 100,
  PRIMARY KEY (org_id, agent_id)
);
