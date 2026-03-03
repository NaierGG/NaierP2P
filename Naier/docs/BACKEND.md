# Backend

## Summary

The backend is a Go 1.22 service built around one deployable process with internal package boundaries.
It currently exposes:

- REST APIs under `/api/v1`
- a websocket endpoint under `/api/v1/ws`
- federation endpoints under `/_federation/v1`

The immediate backend roadmap is:

- split the current mixed key model into identity keys and device keys
- move refresh handling to explicit device sessions
- add durable sync primitives for reconnect and offline recovery
- harden federation with replay protection and shadow users

## Package Map

### `internal/auth`

Responsibilities:

- challenge issuance
- registration and login
- JWT issuance and validation
- profile and device APIs
- backup and device approval flows

Target state:

- identity and device key registration
- device-scoped login
- device trust state
- device session revocation

### `internal/channel`

Responsibilities:

- channel CRUD
- membership management
- DM creation
- invite handling

Target state:

- stable channel membership
- last-read sequence pointer per member
- federation-aware membership sync

### `internal/message`

Responsibilities:

- message persistence
- reactions
- edit and delete
- pagination

Target state:

- `client_event_id`
- `server_event_id`
- monotonic sequence per channel or stream
- message delivery tracking per device

### `internal/websocket`

Responsibilities:

- connection lifecycle
- channel fan-out
- user fan-out across devices
- event routing

Target state:

- websocket as low-latency transport only
- sync endpoint as the authoritative recovery path
- ack-safe reconnect behavior

### `internal/presence`

Responsibilities:

- presence state
- typing state

Target state:

- keep presence ephemeral
- keep read and delivery semantics out of presence

### `internal/media`

Responsibilities:

- upload validation
- object storage interaction
- presigned access

Target state:

- media referenced from encrypted messages
- federation-aware proxy or signed relay rules

### `internal/federation`

Responsibilities:

- remote server discovery
- event signing and verification
- remote user lookup

Target state:

- replay protection
- idempotent event processing
- remote user shadow cache
- allowlist-based rollout

## Current API Surface

### Auth

- `POST /api/v1/auth/challenge`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/me`
- `PUT /api/v1/auth/me`
- `GET /api/v1/auth/devices`
- `DELETE /api/v1/auth/devices/:id`

### Channels

- `POST /api/v1/channels`
- `GET /api/v1/channels`
- `GET /api/v1/channels/:id`
- `PUT /api/v1/channels/:id`
- `DELETE /api/v1/channels/:id`
- `POST /api/v1/channels/join`
- `POST /api/v1/channels/:id/invite`
- `GET /api/v1/channels/:id/members`
- `DELETE /api/v1/channels/:id/members/:userId`
- `POST /api/v1/dm/:userId`

### Messages

- `GET /api/v1/channels/:id/messages`
- `POST /api/v1/channels/:id/messages`
- `PUT /api/v1/messages/:id`
- `DELETE /api/v1/messages/:id`
- `POST /api/v1/messages/:id/reactions`
- `DELETE /api/v1/messages/:id/reactions/:emoji`

### Media

- `POST /api/v1/media/upload`
- `GET /api/v1/media/*objectPath`

### Federation

- `POST /_federation/v1/events`
- `GET /_federation/v1/users/:username`
- `GET /_federation/v1/server-key`
- `GET /_federation/v1/.well-known`

## Planned Backend Contract Changes

### Auth contract

The current `public_key` field is a compatibility shim and must stop being the primary model.

New user-level public fields:

- `identity_signing_key`
- `identity_exchange_key`

New device-level public fields:

- `device_signing_key`
- `device_exchange_key`
- `trusted`
- `approved_by_device_id`
- `revoked_at`

Planned auth additions:

- `POST /api/v1/auth/devices/approve`
- `POST /api/v1/auth/backup/export`
- `POST /api/v1/auth/backup/import`

### Sync contract

The current websocket model is insufficient for durable recovery. The backend will add:

- `client_event_id` on client-originated message events
- `server_event_id` on emitted events
- monotonic `sequence`
- `GET /api/v1/events/sync?after=...`
- `GET /api/v1/channels/:id/sync?after=...`

Read state will move from per-message ack semantics toward:

- channel-level last-read pointer
- delivery state per device

## Data Model Changes

### Existing compatibility requirements

The backend must keep current routes working where practical while new fields are introduced.

Compatibility strategy:

- keep `users.public_key` during migration
- keep `devices.device_key` during migration
- emit both old and new DTO fields temporarily where needed
- add deprecation notes in docs and changelog

### New tables and columns

Planned additions:

- `users.identity_signing_key`
- `users.identity_exchange_key`
- `devices.device_signing_key`
- `devices.device_exchange_key`
- `devices.trusted`
- `devices.approved_by_device_id`
- `devices.revoked_at`
- `device_sessions`
- `message_deliveries`
- `event_offsets`

## Security Rules

Backend rules that must remain true:

- never store private keys
- never store plaintext message bodies by design
- never log token bodies, signatures, raw keys, or ciphertext
- revoke refresh capability at the device-session layer
- reject replayed federation events
- treat websocket as optimization, not as the only source of truth

## Gaps Still Open

These are known gaps that the current codebase still needs to close:

- mixed key model in auth DTOs and client code
- login challenges bound only to username
- device rows behaving like sessions instead of trusted devices
- no durable sync offsets
- no durable per-device delivery tracking
- no federation replay protection or dedupe store

## Near-term acceptance criteria

The backend is considered ready for the next product stage when it can:

1. register a user with identity keys and a first device
2. log in with a device signature, not a user-signature shim
3. approve and revoke devices safely
4. return trustworthy device state to web and mobile clients
5. recover missed events after websocket reconnect
6. process the same federation event only once
