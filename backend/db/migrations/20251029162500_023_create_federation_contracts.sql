-- +goose Up
CREATE TABLE IF NOT EXISTS federation_contracts (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  counterparty_org_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  scope jsonb NOT NULL,
  active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now()
);
-- +goose Down
DROP TABLE IF EXISTS federation_contracts;
