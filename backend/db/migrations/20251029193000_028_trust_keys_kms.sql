-- +goose Up
ALTER TABLE trust_keys
  ALTER COLUMN ed25519_private_key_base64 DROP NOT NULL;
ALTER TABLE trust_keys
  ADD COLUMN IF NOT EXISTS provider text,
  ADD COLUMN IF NOT EXISTS key_ref text,
  ADD COLUMN IF NOT EXISTS key_version text,
  ADD COLUMN IF NOT EXISTS provider_config jsonb DEFAULT '{}'::jsonb,
  ADD COLUMN IF NOT EXISTS jwk_pub jsonb;

-- +goose Down
ALTER TABLE trust_keys
  DROP COLUMN IF EXISTS jwk_pub,
  DROP COLUMN IF EXISTS provider_config,
  DROP COLUMN IF EXISTS key_version,
  DROP COLUMN IF EXISTS key_ref,
  DROP COLUMN IF EXISTS provider;
-- Note: ed25519_private_key_base64 remains nullable after down to avoid data loss