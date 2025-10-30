-- Add a uniqueness constraint to prevent duplicate messages by hash
CREATE UNIQUE INDEX IF NOT EXISTS uniq_fed_gossip_hash ON federation_gossip(hash);

-- Down
-- DROP INDEX IF EXISTS uniq_fed_gossip_hash;
