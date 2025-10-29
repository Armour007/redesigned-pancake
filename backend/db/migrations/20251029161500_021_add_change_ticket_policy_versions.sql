-- +goose Up
ALTER TABLE policy_versions ADD COLUMN IF NOT EXISTS change_ticket text;
-- +goose Down
ALTER TABLE policy_versions DROP COLUMN IF EXISTS change_ticket;
