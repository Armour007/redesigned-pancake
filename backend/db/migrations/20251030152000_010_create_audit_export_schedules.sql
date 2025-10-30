-- Create audit export schedules table
CREATE TABLE IF NOT EXISTS audit_export_schedules (
    org_id UUID PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    cron TEXT NOT NULL,
    dest_type TEXT NOT NULL CHECK (dest_type IN ('webhook','file')),
    dest TEXT NOT NULL,
    format TEXT NOT NULL CHECK (format IN ('json','csv')),
    lookback TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Helpful index when listing
CREATE INDEX IF NOT EXISTS idx_audit_export_schedules_updated ON audit_export_schedules(updated_at DESC);
