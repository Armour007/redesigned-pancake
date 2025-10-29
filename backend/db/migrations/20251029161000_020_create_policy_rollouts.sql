-- +goose Up
CREATE TABLE IF NOT EXISTS policy_rollouts (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  policy_id uuid NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
  version int NOT NULL,
  percent int NOT NULL CHECK (percent >= 0 AND percent <= 100),
  active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_policy_rollouts_org_policy ON policy_rollouts(org_id, policy_id) WHERE active = true;

-- +goose Down
DROP TABLE IF EXISTS policy_rollouts;
