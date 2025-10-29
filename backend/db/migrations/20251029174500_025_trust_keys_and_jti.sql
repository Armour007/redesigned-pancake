-- +goose Up
CREATE TABLE IF NOT EXISTS trust_keys (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  alg text NOT NULL DEFAULT 'EdDSA',
  ed25519_private_key_base64 text NOT NULL,
  kid text,
  active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS trust_keys_org_active ON trust_keys(org_id) WHERE active = true;

CREATE TABLE IF NOT EXISTS trust_token_jti (
  org_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  jti text NOT NULL,
  exp_at timestamptz NOT NULL,
  PRIMARY KEY(org_id, jti)
);

-- +goose Down
DROP TABLE IF EXISTS trust_token_jti;
DROP TABLE IF EXISTS trust_keys;
