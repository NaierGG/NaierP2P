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

### Planned compatibility work in progress

- split user identity keys from device keys
- add device sessions and trust state
- add sync primitives for reconnect-safe delivery
- widen sync beyond message creation to edits, deletes, reactions, and read state
