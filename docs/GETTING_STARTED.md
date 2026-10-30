# Getting started with Linkspire

This guide covers **everything you need** to run Linkspire on your machine: full local stack (Docker) or **hybrid** mode (dashboard on Vercel, backend on your laptop).

**Repository:** [github.com/DanishBuilds/linkspire](https://github.com/DanishBuilds/linkspire)

---

## Prerequisites

| Tool | Used for |
|------|----------|
| [Docker Desktop](https://www.docker.com/products/docker-desktop/) | CockroachDB, Redis, Redpanda, ClickHouse, Go services |
| [Git](https://git-scm.com/) | Clone this repo |
| [Node.js 20+](https://nodejs.org/) (optional) | Fast UI iteration with `npm run dev` in `web/` |
| [cloudflared](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/) | Hybrid only — HTTPS tunnels to localhost |
| [Vercel CLI](https://vercel.com/docs/cli) | Hybrid only — deploy the Next.js dashboard |

**Windows:** run the commands below in **PowerShell** from the **repo root** (the folder that contains `deploy/` and `web/`).

**Secrets:** never commit `deploy/docker/.env` or `web/.env.local`. They are gitignored. Use the `*.example` files as templates.

---

## Option A — Everything local (simplest)

All services, including the dashboard, run in Docker.

### 1. Clone and start

```powershell
git clone https://github.com/DanishBuilds/linkspire.git
cd linkspire
docker compose -f deploy/docker/compose.yaml up --build -d
```

First start can take several minutes (images, migrations, health checks).

### 2. Open the app

| What | URL |
|------|-----|
| Dashboard & marketing | http://localhost:3000 |
| Management API | http://localhost:8081 |
| Short links | http://localhost:8080/{code} |
| CockroachDB admin UI | http://localhost:8090 |

### 3. Register and create a link

1. Go to http://localhost:3000 → **Register** (email + password).
2. **Links** → create a link; the short URL will use `http://localhost:8080/...`.

### 4. After UI changes

```powershell
docker compose -f deploy/docker/compose.yaml build web
docker compose -f deploy/docker/compose.yaml up -d --no-deps --force-recreate web
```

### 5. Frontend-only dev (optional)

```powershell
cd web
copy .env.example .env.local
npm install
npm run dev
```

Stop the `web` container if port **3000** is already in use.

---

## Option B — Hybrid (Vercel dashboard + laptop backend)

Use this when you want a **public HTTPS dashboard** (e.g. `*.vercel.app`) while **API, redirect, workers, and databases** stay on your PC.

```text
Browser → https://your-app.vercel.app     (Next.js on Vercel)
              └─ server calls → https://api-tunnel… → laptop :8081

Click short link → https://redirect-tunnel…/code
              └─ laptop :8080 → destination URL
```

Vercel **cannot** reach `localhost` on your machine. You must expose **8081** (API) and **8080** (redirect) with HTTPS tunnels.

### 1. Start the backend (no web container)

```powershell
git clone https://github.com/DanishBuilds/linkspire.git
cd linkspire
powershell -ExecutionPolicy Bypass -File deploy/scripts/hybrid-backend.ps1
```

Or manually:

```powershell
copy deploy\docker\env.hybrid.example deploy\docker\.env
docker compose -f deploy/docker/compose.backend.yaml --env-file deploy/docker/.env up --build -d
```

Verify locally:

- API: http://localhost:8081/healthz → `ok`
- Redirect: http://localhost:8080 (404 on `/` is normal)

### 2. Install cloudflared (once)

```powershell
winget install Cloudflare.cloudflared
```

### 3. Start tunnels and sync env (automated)

This script starts two quick tunnels, writes `deploy/docker/.env`, and restarts API/redirect/workers:

```powershell
powershell -ExecutionPolicy Bypass -File deploy/scripts/hybrid-tunnels-start.ps1
```

It writes current URLs to `deploy/hybrid/tunnels.current.json` (gitignored).

**Important:** Cloudflare **quick tunnel** URLs **change every time** you restart `cloudflared`. After each restart you must update Vercel (step 4).

Manual alternative: two terminals — `cloudflared tunnel --url http://localhost:8081` and `cloudflared tunnel --url http://localhost:8080`. See [deploy/hybrid/README.md](../deploy/hybrid/README.md).

### 4. Deploy dashboard to Vercel

**One command** (reads tunnel URLs and sets Vercel env + production deploy):

```powershell
powershell -ExecutionPolicy Bypass -File deploy/scripts/hybrid-vercel-deploy.ps1
```

Requires `vercel login` once from `web/`.

**Manual env** (Vercel project → Settings → Environment Variables, **Production**):

| Variable | Value |
|----------|--------|
| `LINKSPIRE_API_URL` | HTTPS tunnel to **8081**, no trailing slash |
| `NEXT_PUBLIC_SHORT_BASE_URL` | HTTPS tunnel to **8080**, no trailing slash |
| `SESSION_COOKIE_SECURE` | `true` |

Then:

```powershell
cd web
vercel link
vercel deploy --prod
```

**Git deploy:** connect the GitHub repo in Vercel with **Root Directory** = `web`, Framework = Next.js. You still must set the three env vars above whenever tunnel URLs change.

### 5. Verify

1. Open your Vercel URL → **Register** / **Login**.
2. Open `https://YOUR-VERCEL-APP.vercel.app/api/health` — expect `"ok":true` and the current API tunnel URL.
3. Create a link; the short URL must use **`NEXT_PUBLIC_SHORT_BASE_URL`**, not `vercel.app`.
4. Open the short link in a private window → redirect should work.

### 6. After reboot, sleep, or Wi‑Fi change

Run **in order** (Docker Desktop must be running):

```powershell
powershell -ExecutionPolicy Bypass -File deploy/scripts/hybrid-backend.ps1
powershell -ExecutionPolicy Bypass -File deploy/scripts/hybrid-tunnels-start.ps1
powershell -ExecutionPolicy Bypass -File deploy/scripts/hybrid-vercel-deploy.ps1
```

Hard refresh the browser (**Ctrl+Shift+R**).

### Hybrid troubleshooting

| Symptom | Likely cause | Fix |
|---------|----------------|-----|
| Login fails / “cannot reach API” | Stale `LINKSPIRE_API_URL` on Vercel | Re-run `hybrid-tunnels-start.ps1` + `hybrid-vercel-deploy.ps1` |
| `/api/health` shows `ok: false` | Tunnels or Docker down | Start Docker, backend script, tunnel script |
| Short link 404 or wrong host | Old redirect tunnel | Update `NEXT_PUBLIC_SHORT_BASE_URL` + `deploy/docker/.env` via tunnel script |
| No analytics in dashboard | ClickHouse pipeline / topic | Ensure full backend stack is up; see README analytics notes |

**Security:** only tunnel **8080** and **8081**. Do not expose database or Kafka ports. Replace dev signing keys in `deploy/docker/.env` if the API is on the public internet (`TOKEN_SIGNING_KEY`, `FEISTEL_KEY`, `COOKIE_SIGNING_KEY`).

---

## Environment reference

| File | Purpose |
|------|---------|
| [deploy/docker/env.hybrid.example](../deploy/docker/env.hybrid.example) | Backend `.env` for hybrid (`PUBLIC_BASE_URL`, `DEFAULT_DOMAIN_HOST`) |
| [web/.env.example](../web/.env.example) | Local `npm run dev` |
| [deploy/docker/compose.yaml](../deploy/docker/compose.yaml) | Full stack including web |
| [deploy/docker/compose.backend.yaml](../deploy/docker/compose.backend.yaml) | Backend only (hybrid) |

---

## Next steps

- Product overview and redirect types: [README.md](../README.md)
- Architecture deep dives: [docs/architecture/](../architecture/)
- OpenAPI: [services/api/openapi.yaml](../services/api/openapi.yaml)
