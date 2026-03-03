DROP INDEX IF EXISTS idx_device_sessions_expires_at;
DROP INDEX IF EXISTS idx_device_sessions_device_id;
DROP TABLE IF EXISTS device_sessions;

DROP INDEX IF EXISTS idx_devices_revoked_at;
DROP INDEX IF EXISTS idx_devices_device_signing_key;

ALTER TABLE devices
  DROP COLUMN IF EXISTS revoked_at,
  DROP COLUMN IF EXISTS approved_by_device_id,
  DROP COLUMN IF EXISTS trusted,
  DROP COLUMN IF EXISTS device_exchange_key,
  DROP COLUMN IF EXISTS device_signing_key;

DROP INDEX IF EXISTS idx_users_identity_signing_key;

ALTER TABLE users
  DROP COLUMN IF EXISTS identity_exchange_key,
  DROP COLUMN IF EXISTS identity_signing_key;
