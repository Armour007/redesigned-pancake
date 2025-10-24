-- +goose Up
CREATE TABLE "agents" (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "organization_id" uuid NOT NULL,
  "name" varchar(255) NOT NULL, -- A human-readable name, e.g., "Client X - Database Backup Script"
  "description" text,
  "created_by_user_id" uuid, -- Can be NULL if the creator user is deleted
  "created_at" timestamptz NOT NULL DEFAULT now(), -- Corrected typo from timamptz
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  -- Foreign key relationships
  FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE CASCADE,
  FOREIGN KEY ("created_by_user_id") REFERENCES "users" ("id") ON DELETE SET NULL
);

-- +goose Down
DROP TABLE "agents";

