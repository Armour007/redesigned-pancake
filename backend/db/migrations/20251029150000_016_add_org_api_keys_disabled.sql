-- Add org setting to disable API keys for verify endpoints
ALTER TABLE organizations
  ADD COLUMN IF NOT EXISTS api_keys_disabled boolean NOT NULL DEFAULT false;

-- No down migration for safety in this prototype