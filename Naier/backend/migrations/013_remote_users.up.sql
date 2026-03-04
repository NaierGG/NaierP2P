CREATE TABLE IF NOT EXISTS remote_users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  remote_user_id TEXT NOT NULL,
  domain TEXT NOT NULL,
  username TEXT NOT NULL,
  display_name TEXT,
  public_key TEXT,
  identity_signing_key TEXT NOT NULL,
  identity_exchange_key TEXT NOT NULL,
  avatar_url TEXT,
  bio TEXT,
  last_synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (domain, username)
);

CREATE INDEX IF NOT EXISTS idx_remote_users_domain_username
  ON remote_users (domain, username);
