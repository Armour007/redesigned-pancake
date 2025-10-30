-- +goose Up
CREATE TABLE IF NOT EXISTS devices (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  device_fingerprint text NOT NULL UNIQUE,
  tpm_ek_pub text,
  tpm_ak_pub text,
  tee_provider text,
  last_attested_at timestamptz,
  posture jsonb,
  posture_ok boolean,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_devices_org ON devices(org_id);

-- +goose Down
DROP TABLE IF EXISTS devices;
