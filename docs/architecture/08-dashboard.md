<!-- docs/architecture/08-dashboard.md -->
# Phase 5 — Dashboard

Runnable code: [`web`](../../web) (Next.js 15 App Router, TypeScript strict, WCAG 2.2 AA).
The dashboard is the control plane for the Phase 4 management API. It ships no analytics
SDKs, no third-party fonts, and no client charting library: everything renders from `self`.

## 1. Stack and shape

| Concern | Choice | Why |
|---------|--------|-----|
| Framework | Next.js 15 (App Router), React 19 | Server Components by default, minimal client JS |
| Language | TypeScript, `strict` + `noUncheckedIndexedAccess` | Catch shape mismatches at build time |
| Styling | Hand-written CSS with design tokens | Full control of contrast, focus, reduced motion; no framework weight |
| Data | Server Components (reads) + Server Actions (writes) | Forms work before hydration; the token never reaches the browser |
| Charts | In-house SVG + accessible data tables | No dependency; guaranteed keyboard and screen-reader paths |
| Validation | `zod` on every action | Server-side input validation that mirrors the API |

Shared JS first load is about 105 kB. Fifteen routes build; pages that read the session cookie
are server-rendered on demand, the rest are static.

## 2. Authentication: a backend-for-frontend

The API issues an HS256 JWT. The browser never sees it. On login or register, a Server Action
stores the token in an **httpOnly, SameSite=Lax** cookie (marked `Secure` in production via
`SESSION_COOKIE_SECURE`). Every API call runs on the Next.js server, which reads that cookie and
attaches the bearer header. An XSS bug cannot exfiltrate the token because no client script can
read an httpOnly cookie.

- `middleware.ts` gates `/dashboard`, `/links`, `/domains`, `/api-keys`, `/webhooks`, `/settings`
  on cookie presence and bounces signed-in users away from `/login` and `/register`.
- Token *validity* is enforced by the API. When `GET /v1/auth/me` returns 401 the app layout
  redirects to `/logout`, which clears the cookie and returns to sign-in (no redirect loop).
- TOTP: if two-factor is on, the sign-in form carries the 6-digit code. A 401 whose detail
  mentions TOTP re-prompts for the code instead of showing a generic failure.

## 3. The API shape problem

Phase 4 serializes two ways: handler DTOs use `snake_case`, while endpoints that return Go repo
structs (domains, API keys, webhooks, DSARs, memberships) emit `PascalCase` because those structs
carry no JSON tags. The dashboard isolates this in one place: `src/lib/api.ts` normalizes every
response to clean camelCase models (`src/lib/types.ts`) by reading both candidate keys. UI code
never sees the inconsistency, and the day the API grows JSON tags, only the normalizers change.

## 4. Accessibility (WCAG 2.2 AA)

The target is AA, including the new 2.2 success criteria.

| Area | What we do |
|------|-----------|
| Structure | One `<h1>` per page, ordered headings, landmarks (`nav`, `main`, `header`), a skip link |
| Keyboard | Everything operable by keyboard; `:focus-visible` is a 3px ring with offset (2.4.11, 2.4.13) |
| Target size | Buttons, inputs, and nav links are at least 44px tall (2.5.8) |
| Color | Status is text plus a shape, never color alone (1.4.1); tokens meet 4.5:1 text and 3:1 UI |
| Forms | Labels tied to controls, `aria-invalid`, errors in `role="alert"`, required fields marked |
| Live regions | Copy-to-clipboard and async results announce via `aria-live` / `role="status"` |
| Charts | A legend, direct labels, and a toggleable data table; the figure has a text summary |
| Motion | `prefers-reduced-motion` collapses all transitions |
| Theming | Light and dark follow `prefers-color-scheme`; both palettes are contrast-checked |

The categorical chart pair (clicks vs unique visitors) was validated for colorblind separation
(OKLab delta E of 29 in light, 23 in dark) and pairs color with dashed strokes, direct legend,
and a data table.

## 5. Privacy in the UI

The dashboard inherits the platform rule that privacy wins ties.

- Analytics shows **privacy-safe daily aggregates only**. There is no IP, no user-agent, and no
  per-visitor drill-down. Unique visitors are labelled "per day" because the salt rotates daily and
  cross-day uniqueness is not computable by design.
- When ClickHouse is not configured the analytics view renders an explicit notice rather than an
  error, matching the API's degrade-to-empty behavior.
- Settings exposes the data-subject rights surface: file a DSAR (access, rectification, erasure,
  restriction, objection, do-not-sell) with the correct GDPR (30-day) or CCPA/CPRA (45-day) window,
  and record a CCPA/CPRA Do Not Sell or Share preference.
- One-time secrets (new API keys, webhook signing secrets) are shown once, masked by default behind
  a reveal toggle, and copyable. They are never persisted client-side.

## 6. Security headers

`next.config.mjs` sets a strict Content-Security-Policy (`default-src 'self'`, `frame-ancestors
'none'`, `object-src 'none'`, `img-src 'self' data:`, no third-party origins), plus
`X-Content-Type-Options`, `Referrer-Policy: no-referrer`, `X-Frame-Options: DENY`, and a
`Permissions-Policy` that denies camera, microphone, and geolocation. The control plane is marked
`noindex`.

## 7. Pages

| Route | Purpose |
|-------|---------|
| `/login`, `/register` | Email/password with region choice and optional TOTP |
| `/dashboard` | Counts (total, active, flagged, active domains) and recent links |
| `/links` | Create (alias, redirect type, password, expiry, click limit) and list |
| `/links/{domainId}/{code}` | Detail, edit, QR request, analytics, delete |
| `/domains` | Add a custom domain and show the exact TXT and CNAME/ALIAS records; state and schedule |
| `/api-keys` | Create (shown once) and revoke scoped keys |
| `/webhooks` | Register signed HTTPS endpoints for `link.created`, `link.flagged`, `domain.verified` |
| `/settings` | Profile, Security (TOTP enrolment with QR), Privacy (DSAR, Do Not Sell) |
| `/report-abuse` | Public, no-auth abuse report into the admin review queue |

Write actions are gated by RBAC in the UI (viewer < member < admin < owner) and again by the API.

## 8. Verified this phase

| Check | Result |
|-------|--------|
| `tsc --noEmit` | clean |
| `next lint` | no warnings or errors |
| `next build` | 15 routes compiled, middleware built, about 105 kB shared JS |
| Boot smoke test | `/login`, `/register`, `/report-abuse` return 200 |
| Auth gate | `/dashboard` with no session returns 307 to `/login?next=/dashboard` |
| Document head | correct `<title>`, `lang="en"`, skip link present |

## 9. Assumptions and open questions

- **Assumptions:** the API runs at `LINKSPIRE_API_URL` (default `http://localhost:8081`); link
  expiry entered via `datetime-local` is interpreted as UTC and sent as RFC 3339; theme follows the
  OS (a manual toggle would need a CSP nonce for its pre-paint script and is deferred).
- **Open questions:** (1) workspace switching UI once a user belongs to several workspaces (the API
  supports `X-Workspace-Id`; the dashboard currently scopes to the first membership). (2) An admin
  abuse-review queue screen (the public report endpoint is wired; the queue lives in `audit_log`).
  (3) QR asset download once the worker writes to object storage in Phase 6 (the request path is
  wired; retrieval is pending the store).
