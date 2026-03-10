# Vercel Setup

This project deploys the web app from `Naier/web` and keeps the API on Fly.io.

## Project Settings

Create a Vercel project from `NaierGG/NaierP2P` and set:

- Framework Preset: `Vite`
- Root Directory: `Naier/web`
- Install Command: `npm ci`
- Build Command: `npm run build`
- Output Directory: `dist`

`web/vercel.json` already rewrites every route to `/index.html`, so client-side routing works without extra configuration.

## Environment Variables

Set these for Production:

```text
VITE_API_BASE_URL=https://api.<domain>/api/v1
VITE_WS_URL=wss://api.<domain>/api/v1/ws
VITE_ENABLE_MOCK_FALLBACK=false
```

Recommended host split:

- `app.<domain>` -> Vercel
- `api.<domain>` -> Fly.io

## Manual First Deploy

1. Import the GitHub repository in Vercel.
2. Set `Root Directory` to `Naier/web`.
3. Add the three production environment variables.
4. Trigger a production deploy.
5. Connect `app.<domain>` after the first successful build.
6. Confirm login, API calls, and WebSocket connection all target `api.<domain>`.

## GitHub Actions Secrets

If you want CI to deploy the web automatically after Fly.io:

```text
WEB_VERCEL_TOKEN
WEB_VERCEL_ORG_ID
WEB_VERCEL_PROJECT_ID
WEB_VITE_API_BASE_URL
WEB_VITE_WS_URL
```

Once these are set, `.github/workflows/deploy.yml` will build and deploy the Vercel app after the backend deploy finishes.
