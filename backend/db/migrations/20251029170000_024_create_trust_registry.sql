-- +goose Up
CREATE TABLE IF NOT EXISTS registry_orgs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name text,
  domain text,
  jwks_url text,
  attestations jsonb DEFAULT '[]'::jsonb,
  status text NOT NULL DEFAULT 'pending',
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS registry_agents (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  agent_id uuid NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
  name text,
  attestations jsonb DEFAULT '[]'::jsonb,
  status text NOT NULL DEFAULT 'pending',
  last_seen_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS registry_agents;
DROP TABLE IF EXISTS registry_orgs;
