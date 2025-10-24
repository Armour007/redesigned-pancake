-- +goose Up
CREATE TABLE "api_keys" (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "organization_id" uuid NOT NULL,
  "name" varchar(255) NOT NULL, -- e.g., "Leila's Primary Key"
  "key_prefix" varchar(8) UNIQUE NOT NULL, -- A short, unique prefix to identify the key (e.g., aura_pk_)
  "hashed_key" text UNIQUE NOT NULL, -- CRITICAL: We ONLY store a secure hash of the key
  "created_by_user_id" uuid, -- Can be NULL if the creator user is deleted
  "last_used_at" timestamptz,
  "expires_at" timestamptz, -- Optional expiry date
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "revoked_at" timestamptz, -- To mark a key as revoked without deleting it
  -- Foreign key relationships
  FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE CASCADE,
  FOREIGN KEY ("created_by_user_id") REFERENCES "users" ("id") ON DELETE SET NULL
);
-- Add an index for faster lookups based on the prefix
CREATE INDEX idx_api_keys_prefix ON api_keys(key_prefix);

-- +goose Down
DROP TABLE "api_keys";

