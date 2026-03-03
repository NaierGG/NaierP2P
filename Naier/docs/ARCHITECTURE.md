# Architecture

## Summary

Naier is a federation-first messenger with web and mobile clients.
The current codebase already has the right high-level boundaries:

- `backend/` for auth, channels, messages, websocket fan-out, presence, media, and federation
- `web/` for the browser client
- `mobile/` for the Flutter client
- `infra/` for local and production deployment

The next architecture step is not a rewrite to pure P2P. The product direction is:

- federation-first
- DM and small trusted groups first
- phone-number-free identity
- strong multi-device support
- offline recovery through server-side store-and-forward
- client-side E2EE with no server plaintext access

## Topology

```text
Web Client ----\
                \
Mobile Client ----> Home Server (Go API + WS + Federation)
                /
Other Servers --/
```

The home server is responsible for:

- authentication
- device session management
- message persistence
- websocket delivery
- offline catch-up sync
- federation ingress and egress

The server is not responsible for:

- storing private keys
- decrypting message bodies
- acting as a global discovery directory in v1

## Trust Model

### User identity keys

Each user has two long-lived identity keys:

- `identity_signing_key` using Ed25519
- `identity_exchange_key` using X25519

These keys define the account identity across devices.

### Device keys

Each device has two device-scoped keys:

- `device_signing_key`
- `device_exchange_key`

The user identity signs the device keys. This creates the trust chain used for:

- login
- new device approval
- device revocation
- future encrypted device-to-device control messages

### Challenge and login

Login is bound to a device, not just a username:

1. client requests a challenge
2. server issues a challenge for a username plus device registration context
3. device signs the challenge
4. server verifies the device signature and the trust chain
5. server issues access and refresh tokens bound to a device session

## Messaging Model

### Realtime path

The websocket hub remains the primary low-latency path for:

- new messages
- edits and deletes
- read state changes
- typing
- reactions

### Consistency path

Websocket alone is not the source of truth. Product-grade consistency comes from sync:

- every client event carries `client_event_id`
- every server event carries `server_event_id`
- ordered events carry a monotonic `sequence`
- each device stores an event offset per stream

This enables:

- reconnect catch-up
- offline device recovery
- duplicate suppression
- stable read and delivery state

## Storage Model

### PostgreSQL

Postgres stores durable state:

- users
- devices
- device sessions
- channels
- channel members
- messages
- message deliveries
- event offsets
- reactions
- federated servers
- federated event dedupe state

### Redis

Redis stores short-lived operational state:

- login challenges
- revoked refresh entries and short-lived session invalidation helpers
- presence
- typing indicators
- websocket pub/sub for multi-instance fan-out

### Client-local storage

The clients store:

- identity private keys
- device private keys
- channel keys
- local search indexes
- encrypted backup payloads before export

## Federation Model

Federation is an extension of the home-server model, not a separate transport stack.

Core rules:

- users are addressed as `@username:domain`
- servers sign federation envelopes
- servers cache remote users as shadow records
- replay protection is mandatory
- duplicate event processing must be idempotent
- remote media is proxied or cached, not blindly trusted

The current repository has federation stubs. The target architecture upgrades that to:

- event dedupe store
- remote user shadowing
- membership sync
- allowlist-based federation rollout

## Anti-abuse Defaults

The product does not begin with a global public network.

Default rules:

- no public directory by default
- invite-only growth
- allowlist-based federation rollout
- rate limits on auth, messaging, and federation ingress
- no requirement for phone number or email

This keeps the initial threat model manageable while preserving the privacy goals.

## Client Priorities

Web and mobile are both first-class clients.

That means every core capability must be defined once and implemented twice:

- auth contract
- device approval flow
- backup import and export
- message send and sync semantics
- read state model

Where platform capabilities differ:

- mobile uses push as a wake-up mechanism and syncs content after launch
- web uses browser notifications and websocket-first delivery

## Short-term Execution Priorities

1. Split identity keys and device keys across the backend, web, and mobile contracts.
2. Add device sessions, trust state, approval flow, and encrypted backup flows.
3. Add sync primitives: `client_event_id`, `server_event_id`, `sequence`, `event_offsets`, and `message_deliveries`.
4. Formalize federation replay protection and remote user shadowing.

## Non-goals for this stage

- pure serverless P2P routing
- blockchain or tokenized anti-abuse
- large public communities
- full MLS rollout before the identity and sync layers are stable
