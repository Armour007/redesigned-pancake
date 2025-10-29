-- +goose Up
CREATE TABLE IF NOT EXISTS policies (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name text NOT NULL,
  engine_type text NOT NULL,
  created_by_user_id uuid,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_policies_org_name ON policies(org_id, name);

-- +goose Down
DROP TABLE IF EXISTS policies;
