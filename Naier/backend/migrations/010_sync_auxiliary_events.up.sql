CREATE SEQUENCE IF NOT EXISTS sync_event_sequence;

SELECT setval(
  'sync_event_sequence',
  COALESCE((SELECT MAX(sequence) FROM messages), 1),
  true
);

ALTER TABLE messages
  ALTER COLUMN sequence SET DEFAULT nextval('sync_event_sequence');

CREATE TABLE IF NOT EXISTS reaction_events (
  event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  sequence BIGINT NOT NULL DEFAULT nextval('sync_event_sequence'),
  message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
  channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  emoji VARCHAR(10) NOT NULL,
  action VARCHAR(10) NOT NULL CHECK (action IN ('add', 'remove')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reaction_events_sequence ON reaction_events(sequence DESC);
CREATE INDEX IF NOT EXISTS idx_reaction_events_channel_sequence ON reaction_events(channel_id, sequence DESC);
CREATE INDEX IF NOT EXISTS idx_reaction_events_message_id ON reaction_events(message_id);

CREATE TABLE IF NOT EXISTS read_events (
  event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  sequence BIGINT NOT NULL DEFAULT nextval('sync_event_sequence'),
  channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  last_read_sequence BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_read_events_sequence ON read_events(sequence DESC);
CREATE INDEX IF NOT EXISTS idx_read_events_channel_sequence ON read_events(channel_id, sequence DESC);
CREATE INDEX IF NOT EXISTS idx_read_events_user_channel ON read_events(user_id, channel_id);
