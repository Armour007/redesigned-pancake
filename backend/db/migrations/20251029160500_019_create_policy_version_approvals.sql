-- +goose Up
CREATE TABLE IF NOT EXISTS policy_version_approvals (
  policy_id uuid NOT NULL,
  version int NOT NULL,
  user_id uuid NOT NULL,
  approved_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (policy_id, version, user_id)
);
CREATE INDEX IF NOT EXISTS idx_policy_version_approvals ON policy_version_approvals(policy_id, version);

-- +goose Down
DROP TABLE IF EXISTS policy_version_approvals;
