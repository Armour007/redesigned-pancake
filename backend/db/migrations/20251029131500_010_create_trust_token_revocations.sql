-- +migrate Up
CREATE TABLE IF NOT EXISTS trust_token_revocations (
  org_id UUID NOT NULL,
  jti TEXT NOT NULL,
  reason TEXT,
  revoked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (org_id, jti)
);

-- +migrate Down
DROP TABLE IF EXISTS trust_token_revocations;
