ALTER TABLE messages
  ADD COLUMN IF NOT EXISTS client_event_id TEXT,
  ADD COLUMN IF NOT EXISTS server_event_id UUID DEFAULT gen_random_uuid(),
  ADD COLUMN IF NOT EXISTS sequence BIGSERIAL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_server_event_id ON messages(server_event_id);
CREATE INDEX IF NOT EXISTS idx_messages_sequence ON messages(sequence DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_sender_client_event
  ON messages(sender_id, client_event_id)
  WHERE client_event_id IS NOT NULL;

ALTER TABLE channel_members
  ADD COLUMN IF NOT EXISTS last_read_sequence BIGINT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS message_deliveries (
  message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
  device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  delivered_at TIMESTAMPTZ,
  read_at TIMESTAMPTZ,
  acked_at TIMESTAMPTZ,
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  PRIMARY KEY (message_id, device_id)
);

CREATE INDEX IF NOT EXISTS idx_message_deliveries_device_id ON message_deliveries(device_id);
CREATE INDEX IF NOT EXISTS idx_message_deliveries_status ON message_deliveries(status);

CREATE TABLE IF NOT EXISTS event_offsets (
  device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  stream_name VARCHAR(64) NOT NULL,
  last_event_id UUID NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (device_id, stream_name)
);

CREATE INDEX IF NOT EXISTS idx_event_offsets_last_event_id ON event_offsets(last_event_id);
