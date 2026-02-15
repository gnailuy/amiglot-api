-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE users
  ALTER COLUMN id SET DEFAULT gen_random_uuid();

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS magic_link_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash BYTEA NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS magic_link_tokens_user_id_idx ON magic_link_tokens(user_id);
CREATE INDEX IF NOT EXISTS magic_link_tokens_expires_at_idx ON magic_link_tokens(expires_at);

-- +goose Down
DROP TABLE IF EXISTS magic_link_tokens;

ALTER TABLE users
  DROP COLUMN IF EXISTS last_login_at;

ALTER TABLE users
  ALTER COLUMN id DROP DEFAULT;
