-- +goose Up
CREATE TABLE IF NOT EXISTS policy_assignments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  policy_id uuid NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
  scope_type text NOT NULL, -- org|team|agent|resource_type (prototype)
  scope_id text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_policy_assignments_scope ON policy_assignments(scope_type, scope_id);
CREATE INDEX IF NOT EXISTS idx_policy_assignments_policy ON policy_assignments(policy_id);

-- +goose Down
DROP TABLE IF EXISTS policy_assignments;
