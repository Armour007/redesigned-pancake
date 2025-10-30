-- +goose Up
CREATE TABLE IF NOT EXISTS org_client_ca (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  kid text,
  algorithm text,
  cert_pem text NOT NULL,
  key_pem text, -- dev/local only - in prod set key_ref and leave key_pem NULL
  key_ref text,
  active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_org_client_ca_active ON org_client_ca(org_id) WHERE active = true;

-- +goose Down
DROP TABLE IF EXISTS org_client_ca;
