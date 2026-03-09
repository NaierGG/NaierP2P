CREATE TABLE beta_invites (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  code        VARCHAR(32) UNIQUE NOT NULL,
  note        TEXT,
  created_by  TEXT NOT NULL,
  max_uses    INT NOT NULL DEFAULT 1,
  use_count   INT NOT NULL DEFAULT 0,
  expires_at  TIMESTAMPTZ,
  disabled_at TIMESTAMPTZ,
  last_used_at TIMESTAMPTZ,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE beta_invite_redemptions (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  invite_id   UUID NOT NULL REFERENCES beta_invites(id) ON DELETE CASCADE,
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  code        VARCHAR(32) NOT NULL,
  redeemed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (invite_id, user_id),
  UNIQUE (user_id)
);

CREATE INDEX idx_beta_invites_active ON beta_invites(code, disabled_at, expires_at);
CREATE INDEX idx_beta_invite_redemptions_invite ON beta_invite_redemptions(invite_id, redeemed_at DESC);
