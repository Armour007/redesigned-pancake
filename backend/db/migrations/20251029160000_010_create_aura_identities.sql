-- +migrate Up
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

-- +migrate Down
DROP TABLE IF EXISTS aura_identities;