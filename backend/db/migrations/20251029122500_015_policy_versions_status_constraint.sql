-- Add CHECK constraint for policy_versions.status to allow 'draft','approved','active'
-- and set default to 'draft' if not already set.

-- Set default to 'draft' for status
ALTER TABLE policy_versions
    ALTER COLUMN status SET DEFAULT 'draft';

-- Add the CHECK constraint (idempotent: migrator ignores "already exists" errors)
ALTER TABLE policy_versions
    ADD CONSTRAINT policy_versions_status_chk
    CHECK (status IN ('draft','approved','active'));
