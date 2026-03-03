DROP INDEX IF EXISTS idx_event_offsets_last_event_id;
DROP TABLE IF EXISTS event_offsets;

DROP INDEX IF EXISTS idx_message_deliveries_status;
DROP INDEX IF EXISTS idx_message_deliveries_device_id;
DROP TABLE IF EXISTS message_deliveries;

ALTER TABLE channel_members
  DROP COLUMN IF EXISTS last_read_sequence;

DROP INDEX IF EXISTS idx_messages_sender_client_event;
DROP INDEX IF EXISTS idx_messages_sequence;
DROP INDEX IF EXISTS idx_messages_server_event_id;

ALTER TABLE messages
  DROP COLUMN IF EXISTS sequence,
  DROP COLUMN IF EXISTS server_event_id,
  DROP COLUMN IF EXISTS client_event_id;
