<!-- web/README.md -->
# Linkspire Dashboard (Phase 5)

The control plane for Linkspire: a Next.js 15 (App Router) dashboard built to WCAG 2.2 AA.
It talks to the Phase 4 management API through a backend-for-frontend so the API token lives
only in an httpOnly cookie and never reaches the browser.

See [`../docs/architecture/08-dashboard.md`](../docs/architecture/08-dashboard.md) for the design.

## Requirements

- Node.js 20 or newer (built and tested on Node 22)
- A running Phase 4 API (`services/api`), by default at `http://localhost:8081`

## Setup

```bash
cp .env.example .env.local   # then edit values
npm install
npm run dev                  # http://localhost:3000
```

## Environment

| Variable | Default | Notes |
|----------|---------|-------|
| `LINKSPIRE_API_URL` | `http://localhost:8081` | Server-side only; the browser never calls the API directly |
| `NEXT_PUBLIC_SHORT_BASE_URL` | `https://lnk.linkspire.io` | Display only (the redirect origin) |
| `SESSION_COOKIE_NAME` | `linkspire_session` | Name of the httpOnly session cookie |
| `SESSION_COOKIE_SECURE` | `false` | Set `true` in production (HTTPS) |

## Scripts

```bash
npm run dev         # development server
npm run build       # production build
npm run start       # serve the production build
npm run lint        # eslint (next/core-web-vitals)
npm run typecheck   # tsc --noEmit
```

## Layout

```text
web/
├── src/
│   ├── app/
│   │   ├── (auth)/           login, register
│   │   ├── (app)/            dashboard, links, domains, api-keys, webhooks, settings
│   │   ├── actions/          server actions (auth, links, resources, account, abuse)
│   │   ├── report-abuse/     public, no-auth
│   │   ├── logout/           cookie-clearing route handler
│   │   ├── layout.tsx        root (html, metadata)
│   │   ├── globals.css       design tokens, a11y, components
│   │   └── not-found.tsx
│   ├── components/           icons, badges, alerts, forms, charts
│   ├── lib/                  api client, types, session, analytics, formatting
│   └── middleware.ts         route gate
└── next.config.mjs           strict CSP + security headers
```
