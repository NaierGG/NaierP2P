# Naier

Naier is a federated messenger prototype with:

- `backend/`: Go API, auth, channels, messages, WebSocket, media, federation
- `web/`: React + Vite + TypeScript client
- `mobile/`: Flutter client
- `infra/`: deployment and reverse-proxy assets
- `docs/`: architecture, backend, web, mobile, federation, deployment, and run guides

## Current Status

The repository is past the scaffold stage.

- Backend core flows are implemented
- Web client is usable for local testing
- Federation receive, shadow state, push, and pull are implemented
- Local two-server federation validation is complete
- Device approval and encrypted backup flows exist on web
- Regression tests cover key auth and federation helpers

Current release target:

- web-only closed beta
- invite-only registration
- static web hosting + Fly.io backend

Still pending for full production-readiness:

- mobile runtime validation on real Flutter toolchains
- broader end-to-end tests
- additional operational hardening and deployment validation

## Quick Start

Backend:

```powershell
cd "c:\Users\KANG HEE\OneDrive\코딩\P2P Messenger\Naier\backend"
docker compose up -d postgres redis postgres-remote redis-remote
docker compose run --rm --entrypoint /app/migrate backend up
docker compose run --rm --entrypoint /app/migrate backend-remote up
docker compose up -d --build backend backend-remote
```

For invite-only local validation:

```powershell
$env:MESH_BETA_INVITE_ONLY="true"
$env:MESH_ADMIN_API_TOKEN="development-admin-token"
docker compose up -d --build backend backend-remote
```

Web:

```powershell
cd "c:\Users\KANG HEE\OneDrive\코딩\P2P Messenger\Naier\web"
npm.cmd install
npm.cmd run build
npm.cmd run preview -- --host 0.0.0.0 --port 4173
```

Health checks:

```powershell
curl.exe http://localhost:8080/health
curl.exe http://localhost:8081/health
```

## Main Docs

- [Run Local](./docs/RUN_LOCAL.md)
- [Testing Checklist](./docs/TESTING.md)
- [Closed Beta Runbook](./docs/BETA_RUNBOOK.md)
- [Architecture](./docs/ARCHITECTURE.md)
- [Backend](./docs/BACKEND.md)
- [Web](./docs/WEB.md)
- [Mobile](./docs/MOBILE.md)
- [Federation](./docs/FEDERATION.md)
- [Deployment](./docs/DEPLOYMENT.md)

## Closed Beta Notes

- Production builds require `VITE_API_BASE_URL`.
- Production builds must set `VITE_ENABLE_MOCK_FALLBACK=false`.
- Release mode backend boot now fails if JWT secret, MinIO credentials, federation keys, or allowed origins are missing or left at defaults.
- Invite management uses the admin API guarded by `X-Admin-Token`.

## Local Endpoints

- Web app: `http://localhost:4173`
- Local backend: `http://localhost:8080`
- Remote federation test backend: `http://localhost:8081`

## Notes

- iPhone runtime testing requires a Mac with Xcode.
- Android emulator should use `10.0.2.2` instead of `localhost` for backend access.
- The backend docker compose includes a second local backend for federation testing.
