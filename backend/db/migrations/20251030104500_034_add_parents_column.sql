-- Add parents column for DAG/CRDT merges (nullable, text[])
ALTER TABLE federation_gossip ADD COLUMN IF NOT EXISTS parents text[] NULL;

-- Index to speed up parents containment checks (optional, GIN)
-- CREATE INDEX IF NOT EXISTS idx_fed_gossip_parents ON federation_gossip USING GIN (parents);

-- Down
-- ALTER TABLE federation_gossip DROP COLUMN IF EXISTS parents;
