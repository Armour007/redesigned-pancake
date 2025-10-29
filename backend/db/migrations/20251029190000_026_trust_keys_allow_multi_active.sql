-- +goose Up
-- Allow multiple active trust keys per org for seamless rotation overlaps
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'trust_keys_org_active'
    ) THEN
        EXECUTE 'DROP INDEX IF EXISTS trust_keys_org_active';
    END IF;
END$$;

-- Non-unique supporting index for active lookups
CREATE INDEX IF NOT EXISTS trust_keys_org_active_idx ON trust_keys(org_id) WHERE active = true;

-- +goose Down
-- Restore uniqueness (single active per org)
DROP INDEX IF EXISTS trust_keys_org_active_idx;
CREATE UNIQUE INDEX IF NOT EXISTS trust_keys_org_active ON trust_keys(org_id) WHERE active = true;
