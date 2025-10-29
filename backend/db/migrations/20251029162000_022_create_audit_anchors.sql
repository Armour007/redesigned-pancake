-- +goose Up
CREATE TABLE IF NOT EXISTS audit_anchors (
  org_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  anchor_date date NOT NULL,
  root_hash text NOT NULL,
  external_ref text,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (org_id, anchor_date)
);
-- +goose Down
DROP TABLE IF EXISTS audit_anchors;
