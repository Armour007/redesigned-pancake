-- +goose Up
CREATE TABLE IF NOT EXISTS trust_tuples (
  id bigserial PRIMARY KEY,
  object_type text NOT NULL,
  object_id text NOT NULL,
  relation text NOT NULL,
  subject_type text NOT NULL,
  subject_id text NOT NULL,
  caveat_json jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_trust_obj_rel ON trust_tuples(object_type, object_id, relation);
CREATE INDEX IF NOT EXISTS idx_trust_subj_rel ON trust_tuples(subject_type, subject_id, relation);

-- +goose Down
DROP TABLE IF EXISTS trust_tuples;
