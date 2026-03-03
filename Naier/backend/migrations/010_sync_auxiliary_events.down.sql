DROP TABLE IF EXISTS read_events;
DROP TABLE IF EXISTS reaction_events;

ALTER TABLE messages
  ALTER COLUMN sequence DROP DEFAULT;

DROP SEQUENCE IF EXISTS sync_event_sequence;
