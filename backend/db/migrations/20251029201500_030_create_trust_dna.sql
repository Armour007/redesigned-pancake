-- +migrate Up
CREATE TABLE IF NOT EXISTS trust_dna (
    id UUID PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    agent_id UUID NULL REFERENCES agents(id) ON DELETE SET NULL,
    owner TEXT NULL,
    vector DOUBLE PRECISION[] NOT NULL,
    dim INTEGER NOT NULL,
    norm DOUBLE PRECISION NOT NULL,
    fingerprint TEXT NOT NULL,
    opt_in BOOLEAN NOT NULL DEFAULT FALSE,
    meta JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_trust_dna_org ON trust_dna(org_id);
CREATE INDEX IF NOT EXISTS idx_trust_dna_org_owner ON trust_dna(org_id, owner);
CREATE INDEX IF NOT EXISTS idx_trust_dna_optin ON trust_dna(opt_in);

-- +migrate Down
DROP TABLE IF EXISTS trust_dna;
