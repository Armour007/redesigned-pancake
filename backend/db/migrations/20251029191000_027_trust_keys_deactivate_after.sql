-- +goose Up
ALTER TABLE trust_keys ADD COLUMN IF NOT EXISTS deactivate_after timestamptz;
CREATE INDEX IF NOT EXISTS trust_keys_deactivate_after_idx ON trust_keys(deactivate_after) WHERE active = true AND deactivate_after IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS trust_keys_deactivate_after_idx;
ALTER TABLE trust_keys DROP COLUMN IF EXISTS deactivate_after;
