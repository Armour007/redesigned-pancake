-- +migrate Up
-- Ensure table exists for environments that missed the original create due to prior migration failures
CREATE TABLE IF NOT EXISTS aura_identities (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    owner TEXT NOT NULL,
    model_hash TEXT NOT NULL,
    code_hash TEXT NOT NULL,
    build_attestation JSONB,
    vc_jwt TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_aura_identities_org ON aura_identities(org_id);

ALTER TABLE aura_identities
    ADD COLUMN IF NOT EXISTS vc_ldp JSONB;

-- +migrate Down
ALTER TABLE aura_identities
    DROP COLUMN IF EXISTS vc_ldp;
