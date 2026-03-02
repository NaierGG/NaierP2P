CREATE TABLE channels (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  type VARCHAR(20) NOT NULL CHECK (type IN ('dm', 'group', 'public')),
  name VARCHAR(100),
  description TEXT,
  invite_code VARCHAR(20) UNIQUE,
  owner_id UUID REFERENCES users(id) ON DELETE SET NULL,
  is_encrypted BOOLEAN NOT NULL DEFAULT TRUE,
  max_members INT NOT NULL DEFAULT 1000 CHECK (max_members > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_channels_owner_id ON channels(owner_id);
CREATE INDEX idx_channels_invite_code ON channels(invite_code);
CREATE INDEX idx_channels_type ON channels(type);
