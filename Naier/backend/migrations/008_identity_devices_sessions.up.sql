ALTER TABLE users
  ADD COLUMN IF NOT EXISTS identity_signing_key TEXT,
  ADD COLUMN IF NOT EXISTS identity_exchange_key TEXT;

UPDATE users
SET
  identity_signing_key = COALESCE(NULLIF(identity_signing_key, ''), public_key),
  identity_exchange_key = COALESCE(NULLIF(identity_exchange_key, ''), public_key)
WHERE identity_signing_key IS NULL
   OR identity_exchange_key IS NULL;

ALTER TABLE users
  ALTER COLUMN identity_signing_key SET NOT NULL,
  ALTER COLUMN identity_exchange_key SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_users_identity_signing_key ON users(identity_signing_key);

ALTER TABLE devices
  ADD COLUMN IF NOT EXISTS device_signing_key TEXT,
  ADD COLUMN IF NOT EXISTS device_exchange_key TEXT,
  ADD COLUMN IF NOT EXISTS trusted BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS approved_by_device_id UUID REFERENCES devices(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMPTZ;

UPDATE devices
SET
  device_signing_key = COALESCE(NULLIF(device_signing_key, ''), device_key),
  device_exchange_key = COALESCE(NULLIF(device_exchange_key, ''), device_key),
  trusted = TRUE
WHERE device_signing_key IS NULL
   OR device_exchange_key IS NULL;

ALTER TABLE devices
  ALTER COLUMN device_signing_key SET NOT NULL,
  ALTER COLUMN device_exchange_key SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_devices_device_signing_key ON devices(device_signing_key);
CREATE INDEX IF NOT EXISTS idx_devices_revoked_at ON devices(revoked_at);

CREATE TABLE IF NOT EXISTS device_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  refresh_jti TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_device_sessions_device_id ON device_sessions(device_id);
CREATE INDEX IF NOT EXISTS idx_device_sessions_expires_at ON device_sessions(expires_at);
