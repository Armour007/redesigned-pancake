-- +goose Up
-- Persisted schedules for audit export scheduler
CREATE TABLE IF NOT EXISTS audit_export_schedules (
    org_id uuid PRIMARY KEY,
    cron text NOT NULL,
    dest_type text NOT NULL,
    dest text NOT NULL,
    format text NOT NULL,
    lookback text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS audit_export_schedules;
