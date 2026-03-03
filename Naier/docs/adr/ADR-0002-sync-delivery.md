# ADR-0002: Sync and Delivery Model

## Status

Accepted

## Context

The current implementation relies on websocket broadcast plus REST pagination.

That is not enough for:

- reconnect recovery
- offline device catch-up
- duplicate suppression
- stable delivered and read state

## Decision

Naier splits low-latency delivery from durable consistency.

### Client-originated idempotency

Every client-originated message event carries `client_event_id`.

The server must treat duplicate `client_event_id` values for the same sender and device as the same logical event.

### Server-side ordering

Every emitted event carries:

- `server_event_id`
- `sequence`

`sequence` is the ordered sync primitive used for read and delivery progress.

### Device sync progress

Each device stores offsets in `event_offsets`:

- `device_id`
- `stream_name`
- `last_event_id`

The sync endpoint uses these offsets to return missed events after reconnect or wake-up.

### Delivery state

Per-device delivery state is tracked in `message_deliveries`:

- `message_id`
- `device_id`
- `delivered_at`
- `read_at`
- `acked_at`
- `status`

### Read state

Read state is tracked at the channel level with a last-read pointer, not by issuing independent read acks for every message.

## Consequences

Positive:

- websocket reconnect becomes safe
- mobile background recovery is simpler
- duplicate sends do not create duplicate messages
- delivered and read semantics become explicit

Negative:

- more storage and write amplification
- more complex event contract
- clients must implement sync and offsets correctly

## API impact

Planned additions:

- `GET /api/v1/events/sync?after=...`
- `GET /api/v1/channels/:id/sync?after=...`

Updated websocket behavior:

- client `MESSAGE_SEND` requires `client_event_id`
- server `MESSAGE_NEW` includes `server_event_id` and `sequence`

## Follow-up

- add message and sync migrations
- update websocket events and handlers
- add integration tests for reconnect and duplicate send scenarios
