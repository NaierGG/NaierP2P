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
- web now has Playwright Chromium smoke coverage for key generation, registration, settings navigation, and new-device onboarding
- web now has a separate `@live` Playwright smoke path for registration and settings navigation against a running backend
- Playwright mock coverage now includes chat send flow and backup-driven device approval end-to-end
- mobile runtime config now auto-selects emulator-friendly API defaults and supports explicit API/WS overrides via dart defines
- backend now supports invite-only beta registration with invite issuance, redemption tracking, and disablement
- backend now exposes admin-only invite management endpoints guarded by `X-Admin-Token`
- release-mode backend boot now validates allowed origins, JWT secret, MinIO credentials, federation keys, and invite admin token requirements
- backend CORS is now allowlist-based via `MESH_SERVER_ALLOWED_ORIGINS`
- web registration now supports invite codes
- web production runtime now requires `VITE_API_BASE_URL`
- web mock fallback is now controlled by `VITE_ENABLE_MOCK_FALLBACK` and defaults to off in production builds
- channel and message loading now surface explicit runtime errors instead of silently falling through in production
- web now ships a Vercel-compatible SPA rewrite file and production env example
- backend and infra now ship production env examples for closed beta deployment
- CI now runs Playwright smoke on every web build and includes a backend integration job placeholder for invite-only/live validation
- live Playwright smoke now covers invite-only registration and real-backend channel message send
- production nginx is now rendered from `API_DOMAIN` and the API compose stack now requires real secrets instead of placeholder defaults
- Fly deployment config now keeps one machine warm to avoid beta cold starts
- GitHub Actions can now optionally deploy the static web app to Vercel when the web deployment secrets are configured
- mobile registration now accepts invite codes and uses runtime platform metadata so invite-only beta servers stay API-compatible with the Flutter client
- deployment docs now include an explicit Vercel setup guide with `Naier/web` root-directory requirements
- GitHub Actions deploy paths now resolve from the real repository root instead of assuming `backend`, `web`, and `mobile` live at the top level

### Planned compatibility work in progress

- split user identity keys from device keys
- add device sessions and trust state
- add sync primitives for reconnect-safe delivery
- widen sync beyond message creation to edits, deletes, reactions, and read state
