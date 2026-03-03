# ADR-0001: Key Model

## Status

Accepted

## Context

The current codebase mixes two different responsibilities into one public key field:

- identity and signature verification
- message key agreement

That is not extensible to:

- trusted multi-device login
- device approval
- backup and recovery
- future encrypted device-to-device control messages

## Decision

Naier uses separate user identity keys and device keys.

### User identity keys

- `identity_signing_key`: Ed25519
- `identity_exchange_key`: X25519

These are long-lived account keys.

### Device keys

- `device_signing_key`
- `device_exchange_key`

These are per-device public keys.

### Trust chain

The user identity signs the device keys.

This means:

- the server can verify that a device belongs to a user
- a device can be revoked without rotating the whole account identity
- new device approval can be represented as a signed trust update

### Login model

Login uses the device signing key over a server challenge.

The challenge is bound to:

- username
- device registration or login context

The server validates:

- device signature
- device trust state
- device session state

## Consequences

Positive:

- clears up the current Ed25519/X25519 confusion
- makes multi-device support explicit
- supports device approval and recovery flows
- aligns web and mobile around one contract

Negative:

- requires schema changes
- requires DTO changes across backend, web, and mobile
- needs compatibility shims while `public_key` still exists

## Compatibility

During migration:

- `users.public_key` remains in storage for compatibility
- DTOs gradually move to explicit identity fields
- old clients can continue working until the new contract is enforced

## Follow-up

- add migrations for user and device key columns
- change register and login contracts
- add device approval and backup flows
