-- +goose Up
CREATE TABLE IF NOT EXISTS client_certs (
  serial text PRIMARY KEY,
  org_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  device_id uuid REFERENCES devices(id) ON DELETE SET NULL,
  subject text,
  cert_pem text NOT NULL,
  not_before timestamptz NOT NULL,
  not_after timestamptz NOT NULL,
  revoked boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_client_certs_org ON client_certs(org_id);

-- +goose Down
DROP TABLE IF EXISTS client_certs;
