-- +goose Up
CREATE TABLE IF NOT EXISTS policy_versions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  policy_id uuid NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
  version int NOT NULL,
  body jsonb NOT NULL,
  compiled_blob bytea,
  checksum text,
  status text NOT NULL DEFAULT 'active', -- draft|active (prototype)
  created_by_user_id uuid,
  created_at timestamptz NOT NULL DEFAULT now(),
  approved_by_user_id uuid,
  approved_at timestamptz,
  activated_at timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_policy_versions ON policy_versions(policy_id, version);
CREATE INDEX IF NOT EXISTS idx_policy_versions_policy ON policy_versions(policy_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS policy_versions;
