# Testing

This document lists the highest-value validation paths for the current Naier repository.

## Automated Checks

Backend:

```powershell
cd "c:\Users\KANG HEE\OneDrive\코딩\P2P Messenger\Naier\backend"
$src = (Get-Location).Path -replace '\\','/'
$src = $src -replace '^C:','//c'
docker run --rm -v "${src}:/src" -w /src golang:1.22-alpine /bin/sh -c "apk add --no-cache git >/dev/null && go test ./..."
```

Web build:

```powershell
cd "c:\Users\KANG HEE\OneDrive\코딩\P2P Messenger\Naier\web"
npm.cmd run build
```

Web browser smoke:

```powershell
cd "c:\Users\KANG HEE\OneDrive\코딩\P2P Messenger\Naier\web"
npm.cmd run test:e2e
```

Web browser live smoke against the real backend:

```powershell
cd "c:\Users\KANG HEE\OneDrive\코딩\P2P Messenger\Naier\web"
npm.cmd run test:e2e:live
```

The `:live` suite checks `http://127.0.0.1:8080/health` first and skips itself if the backend is not running.
For invite-only live runs, also set:

```powershell
$env:PLAYWRIGHT_ADMIN_TOKEN="development-admin-token"
$env:VITE_ENABLE_MOCK_FALLBACK="false"
```

Backend integration smoke:

```powershell
cd "c:\Users\KANG HEE\OneDrive\코딩\P2P Messenger\Naier\backend"
go run ./cmd/integration-smoke
```

## Manual Core Scenarios

### Auth

Verify:

1. Register a new account
2. Log in with challenge flow
3. Open Settings > Security
4. Export encrypted backup
5. Clear local identity
6. Restore encrypted backup

### Device Approval

Verify:

1. Open `/auth/device` in a second browser profile
2. Restore encrypted backup there
3. Copy pairing payload
4. Approve from the original trusted session
5. Complete sign-in in the second browser

### Chat Sync

Verify:

1. Open the same account on two web sessions
2. Send a message
3. Refresh one session
4. Confirm missed events are recovered
5. Confirm `DELIVERY_ACK` and read state continue to work

### Federation

Verify:

1. Start both local backends
2. Confirm `8080` and `8081` health
3. Trigger protected federation sync
4. Trigger protected federation pull
5. Confirm remote shadow rows exist in Postgres

## What Is Already Covered by Regression Tests

Backend regression tests currently cover:

- auth middleware access-token enforcement
- refresh token rejection on protected routes
- malformed claim rejection
- federation TXT record parsing
- federation key decode helpers
- federation sign/verify helper paths

Browser smoke currently covers:

- key generation flow
- registration flow
- app entry and settings navigation
- chat send flow in the app shell
- backup and device approval end-to-end flow
- new-device onboarding page
- live backend invite-only registration and message send when `8080` is available

## Still Worth Adding Later

- message create/edit/delete sync scenarios
- multi-device read-state validation
- mobile runtime smoke tests
