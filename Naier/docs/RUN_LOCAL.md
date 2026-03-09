# Run Local

This guide is the shortest path to running the current Naier stack locally.

## 1. Prerequisites

Required:

- Docker Desktop
- Node.js 20+
- npm

Optional:

- Flutter 3.22+ for mobile validation
- Xcode on macOS for iPhone testing

## 2. Start Backend

From [backend](/c:/Users/KANG%20HEE/OneDrive/%EC%BD%94%EB%94%A9/P2P%20Messenger/Naier/backend):

```powershell
docker compose up -d postgres redis postgres-remote redis-remote
docker compose run --rm --entrypoint /app/migrate backend up
docker compose run --rm --entrypoint /app/migrate backend-remote up
docker compose up -d --build backend backend-remote
```

Check health:

```powershell
curl.exe http://localhost:8080/health
curl.exe http://localhost:8081/health
```

Expected:

- `8080` is the main local backend
- `8081` is the second backend used for federation testing

## 3. Start Web

From [web](/c:/Users/KANG%20HEE/OneDrive/%EC%BD%94%EB%94%A9/P2P%20Messenger/Naier/web):

```powershell
npm.cmd install
npm.cmd run build
npm.cmd run preview -- --host 0.0.0.0 --port 4173
```

Open:

- `http://localhost:4173`

If you want invite-only local registration:

```powershell
cd "c:\Users\KANG HEE\OneDrive\코딩\P2P Messenger\Naier\backend"
$env:MESH_BETA_INVITE_ONLY="true"
$env:MESH_ADMIN_API_TOKEN="development-admin-token"
docker compose up -d --build backend backend-remote
```

Create an invite:

```powershell
curl.exe -X POST http://localhost:8080/api/v1/admin/invites ^
  -H "Content-Type: application/json" ^
  -H "X-Admin-Token: development-admin-token" ^
  -d "{\"note\":\"local beta\",\"max_uses\":1}"
```

## 4. Stop Local Stack

From [backend](/c:/Users/KANG%20HEE/OneDrive/%EC%BD%94%EB%94%A9/P2P%20Messenger/Naier/backend):

```powershell
docker compose down
```

## 5. Useful Backend Commands

View logs:

```powershell
docker compose logs backend --tail 100
docker compose logs backend-remote --tail 100
```

Rebuild backends:

```powershell
docker compose up -d --build backend backend-remote
```

Run migrations again:

```powershell
docker compose run --rm --entrypoint /app/migrate backend up
docker compose run --rm --entrypoint /app/migrate backend-remote up
```

## 6. Mobile Notes

Android emulator backend host:

- `http://10.0.2.2:8080`

iOS simulator backend host:

- `http://localhost:8080`

iPhone physical-device testing:

- requires macOS + Xcode

## 7. Current Local Test Scope

Verified locally:

- backend health
- web build
- two-server federation push
- two-server federation pull
- remote shadow state persistence

Not fully verified in this Windows environment:

- Flutter runtime
- iPhone physical-device run
