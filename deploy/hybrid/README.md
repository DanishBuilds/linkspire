# Hybrid deploy: Vercel (dashboard) + Docker (backend on your laptop)

The **Next.js app** runs on **Vercel**. **API, redirect, workers, and databases** run in **Docker on your PC**. Vercel cannot call `localhost` on your machine — you expose **API (8081)** and **redirect (8080)** with an HTTPS tunnel.

## Architecture

```text
Browser → https://your-app.vercel.app          (dashboard)
              └─ Vercel server → https://api-tunnel…  → laptop :8081

Click short link → https://lnk-tunnel…/code      (redirect)
              └─ tunnel → laptop :8080 → destination
```

## Automated flow (Windows)

From the **repo root**, after each reboot or tunnel restart:

```powershell
powershell -ExecutionPolicy Bypass -File deploy/scripts/hybrid-backend.ps1
powershell -ExecutionPolicy Bypass -File deploy/scripts/hybrid-tunnels-start.ps1
powershell -ExecutionPolicy Bypass -File deploy/scripts/hybrid-vercel-deploy.ps1
```

See also **[docs/GETTING_STARTED.md](../../docs/GETTING_STARTED.md)** (Option B).

## 1. Start backend on the laptop

From the **repo root**:

```powershell
powershell -ExecutionPolicy Bypass -File deploy/scripts/hybrid-backend.ps1
```

Or manually:

```powershell
copy deploy\docker\env.hybrid.example deploy\docker\.env
docker compose -f deploy/docker/compose.backend.yaml --env-file deploy/docker/.env up --build -d
```

Leave Docker running. Check locally:

| Service   | URL |
|-----------|-----|
| API       | http://localhost:8081 |
| Redirect  | http://localhost:8080/{code} |
| CRDB UI   | http://localhost:8090 |

## 2. HTTPS tunnels (required for Vercel)

Install [Cloudflare Tunnel (`cloudflared`)](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/) or use [ngrok](https://ngrok.com/) twice (one process per port).

### Quick test (random URLs each run — fine for trying)

**Terminal A** (API):

```powershell
cloudflared tunnel --url http://localhost:8081
```

Copy the `https://….trycloudflare.com` URL → this is **`LINKSPIRE_API_URL`** for Vercel.

**Terminal B** (redirect):

```powershell
cloudflared tunnel --url http://localhost:8080
```

Copy the URL → set in **`deploy/docker/.env`**:

```env
PUBLIC_BASE_URL=https://….trycloudflare.com
DEFAULT_DOMAIN_HOST=….trycloudflare.com
```

Then restart redirect + workers so QR/short links use the public host:

```powershell
docker compose -f deploy/docker/compose.backend.yaml --env-file deploy/docker/.env up -d --force-recreate redirect workers api
```

### Stable hostnames (recommended)

Use a [named Cloudflare Tunnel](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/configure-tunnels/local-management/) with two public hostnames, e.g. `api.yourdomain.com` → `:8081`, `lnk.yourdomain.com` → `:8080`.

## 3. Deploy dashboard to Vercel

```powershell
cd web
vercel login
vercel link
```

In the [Vercel project → Settings → Environment Variables](https://vercel.com/docs/projects/environment-variables), set for **Production** (and Preview if you want):

| Name | Value |
|------|--------|
| `LINKSPIRE_API_URL` | `https://…` tunnel to **8081** (no trailing slash) |
| `NEXT_PUBLIC_SHORT_BASE_URL` | Same as `PUBLIC_BASE_URL` in `deploy/docker/.env` |
| `SESSION_COOKIE_SECURE` | `true` |

Deploy:

```powershell
vercel deploy --prod
```

Or connect the GitHub repo in the Vercel UI: **Root Directory** = `web`, Framework = Next.js.

## 4. Verify

1. Open the Vercel URL → **Register** / **Login** (Vercel calls your tunneled API).
2. Create a link; copy the short URL — it must use **`NEXT_PUBLIC_SHORT_BASE_URL`**, not `vercel.app`.
3. Open the short URL in a private window → redirect should hit your laptop redirect service.

## Security notes

- Only tunnel **8080** and **8081**. Do not expose Cockroach, Redis, or Redpanda ports on the public internet.
- Replace dev signing keys in `deploy/docker/.env` if the API is on the public internet.
- Laptop sleep or tunnel stopped → dashboard and short links stop working.

## Optional: passkeys / OIDC

If you enable WebAuthn or OIDC on the API, set origins to your **Vercel** URL (`WEBAUTHN_RP_ID`, `WEBAUTHN_RP_ORIGIN`, `OIDC_POST_LOGIN_URL`, etc.) on the **API** container env — see `services/api/internal/config/config.go`.
