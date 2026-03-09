# Deployment

This repository now targets a web-only closed beta release.

## Release Topology

- `app.<domain>`: static web hosting
- `api.<domain>`: Naier Go API on Fly.io
- optional reverse proxy: `infra/nginx/nginx.conf.template`

The web app is not served by the backend nginx stack. Build the web client separately and deploy it to a static host such as Vercel.

## Required Production Environment

Backend:

- `MESH_SERVER_MODE=release`
- `MESH_SERVER_ALLOWED_ORIGINS=https://app.<domain>`
- `MESH_AUTH_JWT_SECRET`
- `MESH_BETA_INVITE_ONLY=true`
- `MESH_ADMIN_API_TOKEN`
- `MESH_MEDIA_MINIO_ENDPOINT`
- `MESH_MEDIA_MINIO_BUCKET`
- `MESH_MEDIA_MINIO_ACCESS_KEY`
- `MESH_MEDIA_MINIO_SECRET_KEY`
- `MESH_FEDERATION_SERVER_DOMAIN`
- `MESH_FEDERATION_SERVER_PUBLIC_KEY`
- `MESH_FEDERATION_SERVER_PRIVATE_KEY`

Examples:

- [backend/.env.production.example](/c:/Users/KANG%20HEE/OneDrive/%EC%BD%94%EB%94%A9/P2P%20Messenger/Naier/backend/.env.production.example)
- [infra/.env.production.example](/c:/Users/KANG%20HEE/OneDrive/%EC%BD%94%EB%94%A9/P2P%20Messenger/Naier/infra/.env.production.example)
- [web/.env.production.example](/c:/Users/KANG%20HEE/OneDrive/%EC%BD%94%EB%94%A9/P2P%20Messenger/Naier/web/.env.production.example)

## Web Deployment

Build the web client with:

```powershell
cd "c:\Users\KANG HEE\OneDrive\코딩\P2P Messenger\Naier\web"
npm.cmd ci
npm.cmd run build
```

Deploy the generated `web/dist` folder to a static host. `web/vercel.json` already includes a SPA rewrite for Vercel-style hosting.

Required web runtime variables:

- `VITE_API_BASE_URL=https://api.<domain>/api/v1`
- `VITE_WS_URL=wss://api.<domain>/api/v1/ws`
- `VITE_ENABLE_MOCK_FALLBACK=false`

Optional GitHub Actions secrets for automatic Vercel deployment:

- `WEB_VERCEL_TOKEN`
- `WEB_VERCEL_ORG_ID`
- `WEB_VERCEL_PROJECT_ID`
- `WEB_VITE_API_BASE_URL`
- `WEB_VITE_WS_URL`

If all five secrets are present, `.github/workflows/deploy.yml` will:

1. pull Vercel production project metadata
2. build the prebuilt web artifact with production API envs
3. deploy the static app after the Fly.io backend deployment succeeds

## Backend Deployment

Fly.io is configured in [infra/fly.toml](/c:/Users/KANG%20HEE/OneDrive/%EC%BD%94%EB%94%A9/P2P%20Messenger/Naier/infra/fly.toml).

Key points:

- app name: `naier-api`
- release command: `/app/migrate up`
- health check: `/health`
- `auto_stop_machines = "off"` to avoid beta cold starts

The CI workflow syncs release secrets before `flyctl deploy`.

## Reverse Proxy

The production compose stack in [infra/docker-compose.yml](/c:/Users/KANG%20HEE/OneDrive/%EC%BD%94%EB%94%A9/P2P%20Messenger/Naier/infra/docker-compose.yml) is API-only.

- nginx proxies `/api/`, `/ws`, and `/_federation/`
- cert paths are rendered from `API_DOMAIN`
- web requests are intentionally not served there

## Closed Beta Operations

Invite-only registration is enforced by the backend when `MESH_BETA_INVITE_ONLY=true`.

Admin invite endpoints:

- `GET /api/v1/admin/invites`
- `POST /api/v1/admin/invites`
- `DELETE /api/v1/admin/invites/:id`

Auth:

- header `X-Admin-Token: <token>`
- optional audit header `X-Admin-Actor: <operator>`

## Pre-Release Checklist

- set real domains and TLS certificates
- confirm `MESH_SERVER_ALLOWED_ORIGINS` matches the web host exactly
- confirm mock fallback is disabled in web production env
- create and redeem a beta invite
- verify backup export/import
- verify device approval
- verify federation state sync against a second server when enabled
- restore a recent PostgreSQL backup into a throwaway environment
