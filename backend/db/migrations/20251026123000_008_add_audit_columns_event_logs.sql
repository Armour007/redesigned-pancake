-- +goose Up
ALTER TABLE event_logs
  ADD COLUMN IF NOT EXISTS request_id text,
  ADD COLUMN IF NOT EXISTS user_agent text,
  ADD COLUMN IF NOT EXISTS path text,
  ADD COLUMN IF NOT EXISTS status_code integer;

-- Optional indexes for searchability
CREATE INDEX IF NOT EXISTS idx_event_logs_path ON event_logs(path);
CREATE INDEX IF NOT EXISTS idx_event_logs_status ON event_logs(status_code);

-- +goose Down
ALTER TABLE event_logs
  DROP COLUMN IF EXISTS request_id,
  DROP COLUMN IF EXISTS user_agent,
  DROP COLUMN IF EXISTS path,
  DROP COLUMN IF EXISTS status_code;
