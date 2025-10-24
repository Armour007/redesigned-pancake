-- +goose Up
CREATE TABLE "organization_members" (
  "organization_id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "role" varchar(50) NOT NULL DEFAULT 'member', -- e.g., 'admin', 'member', 'billing'
  "joined_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("organization_id", "user_id"), -- Composite primary key
  -- Foreign key relationships
  FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE CASCADE,
  FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE CASCADE
);

-- +goose Down
DROP TABLE "organization_members";

