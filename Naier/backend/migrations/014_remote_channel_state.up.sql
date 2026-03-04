CREATE TABLE IF NOT EXISTS remote_channels (
  origin_server TEXT NOT NULL,
  remote_channel_id TEXT NOT NULL,
  channel_type TEXT NOT NULL,
  name TEXT,
  description TEXT,
  is_encrypted BOOLEAN NOT NULL DEFAULT TRUE,
  max_members INT NOT NULL DEFAULT 0,
  member_count INT NOT NULL DEFAULT 0,
  last_synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (origin_server, remote_channel_id)
);

CREATE TABLE IF NOT EXISTS remote_channel_memberships (
  origin_server TEXT NOT NULL,
  remote_channel_id TEXT NOT NULL,
  remote_user_id TEXT NOT NULL,
  username TEXT NOT NULL,
  display_name TEXT,
  role TEXT NOT NULL DEFAULT 'member',
  joined_at TIMESTAMPTZ,
  is_muted BOOLEAN NOT NULL DEFAULT FALSE,
  last_synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (origin_server, remote_channel_id, remote_user_id)
);

CREATE INDEX IF NOT EXISTS idx_remote_channel_memberships_channel
  ON remote_channel_memberships (origin_server, remote_channel_id);
