CREATE TABLE IF NOT EXISTS federated_events (
  event_id TEXT NOT NULL,
  origin_server TEXT NOT NULL,
  event_type TEXT NOT NULL,
  payload_hash TEXT NOT NULL,
  received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  processed_at TIMESTAMPTZ,
  PRIMARY KEY (origin_server, event_id)
);

CREATE INDEX IF NOT EXISTS idx_federated_events_received_at
  ON federated_events (received_at DESC);
