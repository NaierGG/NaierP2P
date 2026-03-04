# Changelog

## Unreleased

### Added

- ADR-0001 for the split identity and device key model
- ADR-0002 for sync and delivery semantics
- rewritten architecture and backend docs aligned to the federation-first roadmap
- pending device registration and approval flow scaffolding for trusted multi-device onboarding
- persisted sync cursors per device stream via `event_offsets`
- auxiliary sync event storage for reactions and channel-level read state
- mobile sync client scaffolding with stored cursors and reconnect catch-up
- message delivery rows are now populated from message fetch/sync flows and read acknowledgements
- mobile channel list/detail screens now prefer real API and WebSocket data with demo fallback
- mobile key generation and challenge login are wired to the backend auth contract
- live `MESSAGE_NEW` broadcasts now mark connected device deliveries immediately
- mobile channel detail now resolves real channel/member metadata for sender names
- clients can send `DELIVERY_ACK`, allowing the server to populate `acked_at`
- web now sends `DELIVERY_ACK` on new message receipt and sync catch-up
- mobile login/register flows now jump directly into the first available channel when possible
- encrypted key-bundle backups can now be exported from the web client with client-side passphrase encryption
- backend now stores opaque encrypted backup blobs via `/api/v1/auth/backup/export` and `/api/v1/auth/backup/import`
- web security settings now supports encrypted backup export, stored-backup retrieval, and local restore
- mobile secure storage now has a dedicated encrypted-backup slot for follow-up recovery UX
- web now has a dedicated `/auth/device` flow for preparing a new device from an encrypted backup and finishing approval-based sign-in
- trusted web sessions can now register and approve a pending device in one flow from security settings
- web security settings now separates pending devices from trusted devices and exposes a direct new-device link for approval-based onboarding
- the new-device page now restores prepared device keys on reload and gives clearer post-approval sign-in guidance
- federation ingress now records `(origin_server, event_id)` with payload hashes to reject duplicate or replayed events with mutated payloads
- federation event acknowledgements now distinguish processed events from accepted duplicates
- federation now persists remote user shadow records and refreshes them from both direct fetches and `USER_SYNC` events
- local federation user lookups now include identity signing and exchange keys in the published payload
- federation now exposes `/_federation/v1/channels/:id/state` and can persist remote channel/member shadow state from `CHANNEL_STATE_SYNC` events
- authenticated local users can now push or pull channel state via `/api/v1/federation/channels/:id/sync` and `/api/v1/federation/channels/:id/pull`
- federation server registry rows can now store explicit endpoints for local and non-DNS federation targets
- local development compose now includes an isolated `backend-remote` federation peer with its own Postgres and Redis services
- resolver now prefers cached/database federation endpoints instead of falling back to DNS-derived defaults during push and pull
- backend now has regression tests for auth access-token enforcement and federation TXT/signature helper paths

### Planned compatibility work in progress

- split user identity keys from device keys
- add device sessions and trust state
- add sync primitives for reconnect-safe delivery
- widen sync beyond message creation to edits, deletes, reactions, and read state
