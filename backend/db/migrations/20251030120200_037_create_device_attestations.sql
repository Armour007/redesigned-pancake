-- +goose Up
CREATE TABLE IF NOT EXISTS device_attestations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  device_id uuid REFERENCES devices(id) ON DELETE CASCADE,
  type text NOT NULL, -- tpm|aws_nitro|azure_snp|intel_tdx|gcp_cc
  raw jsonb NOT NULL,
  verified boolean NOT NULL DEFAULT false,
  verified_at timestamptz,
  error text,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_device_attestations_org_dev ON device_attestations(org_id, device_id);

-- +goose Down
DROP TABLE IF EXISTS device_attestations;
