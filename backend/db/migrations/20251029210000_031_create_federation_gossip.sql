-- Federation gossip tables
CREATE TABLE IF NOT EXISTS federation_peers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url TEXT NOT NULL,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Minimal signed messages (append-only)
-- topic: 'revocation' | 'key_rotation' | 'org_registry'
CREATE TABLE IF NOT EXISTS federation_gossip (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL,
    topic TEXT NOT NULL,
    payload JSONB NOT NULL,
    ts TIMESTAMPTZ NOT NULL,
    nonce TEXT NOT NULL,
    kid TEXT,
    jws TEXT NOT NULL,
    prev_hash TEXT,
    hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_fed_gossip_topic_ts ON federation_gossip(topic, ts);
CREATE INDEX IF NOT EXISTS idx_fed_gossip_hash ON federation_gossip(hash);

-- Down
-- DROP TABLE IF EXISTS federation_gossip;
-- DROP TABLE IF EXISTS federation_peers;
