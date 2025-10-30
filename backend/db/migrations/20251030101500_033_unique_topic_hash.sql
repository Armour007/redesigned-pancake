-- Switch to composite uniqueness scoped by topic
DROP INDEX IF EXISTS uniq_fed_gossip_hash;
CREATE UNIQUE INDEX IF NOT EXISTS uniq_fed_gossip_topic_hash ON federation_gossip(topic, hash);

-- Down
-- DROP INDEX IF EXISTS uniq_fed_gossip_topic_hash;
-- CREATE UNIQUE INDEX IF NOT EXISTS uniq_fed_gossip_hash ON federation_gossip(hash);
