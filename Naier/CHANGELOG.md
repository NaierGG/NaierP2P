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

### Planned compatibility work in progress

- split user identity keys from device keys
- add device sessions and trust state
- add sync primitives for reconnect-safe delivery
- widen sync beyond message creation to edits, deletes, reactions, and read state
