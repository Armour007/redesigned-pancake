-- Drop legacy unique index (safe if not present)
-- Non-unique supporting index for active lookups

-- +goose Down
-- Restore uniqueness (single active per org)
DROP INDEX IF EXISTS trust_keys_org_active_idx;
CREATE UNIQUE INDEX IF NOT EXISTS trust_keys_org_active ON trust_keys(org_id) WHERE active = true;
