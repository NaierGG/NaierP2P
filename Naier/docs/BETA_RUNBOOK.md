# Closed Beta Runbook

This runbook is the minimum operator guide for the web closed beta.

## Issue a Beta Invite

```powershell
curl.exe -X POST https://api.example.com/api/v1/admin/invites ^
  -H "Content-Type: application/json" ^
  -H "X-Admin-Token: <admin-token>" ^
  -H "X-Admin-Actor: operator-name" ^
  -d "{\"note\":\"designer cohort\",\"max_uses\":1}"
```

List active invites:

```powershell
curl.exe https://api.example.com/api/v1/admin/invites ^
  -H "X-Admin-Token: <admin-token>"
```

Disable an invite:

```powershell
curl.exe -X DELETE https://api.example.com/api/v1/admin/invites/<invite-id> ^
  -H "X-Admin-Token: <admin-token>"
```

## User Onboarding Flow

1. Send the invite code.
2. Tell the user to open `app.<domain>`.
3. Ask them to generate keys and store the encrypted backup immediately.
4. Confirm they can enter the app after registration.
5. For second-device setup, direct them to `/auth/device`.

## Support: Common Problems

### Registration rejected

Check:

- invite not expired
- invite not disabled
- invite still has remaining uses
- browser is pointed at the production API domain

### User lost browser profile

Recovery path:

1. User must have the encrypted backup blob and passphrase.
2. Restore through the security settings or `/auth/device`.
3. If a previous trusted device still exists, approve the new device from that session.

### API outage suspicion

Check:

```powershell
curl.exe https://api.example.com/health
```

If health fails:

- confirm Fly deployment state
- inspect backend logs
- confirm database and redis connectivity
- do not rely on mock fallback in production

## Pre-Beta Daily Smoke

1. Create an invite.
2. Register a fresh account.
3. Send a message.
4. Export an encrypted backup.
5. Approve a second device.
6. Verify `/health`.
