<!-- docs/architecture/07-management-api.md -->
# Phase 4 — Management API and Workers

Runnable code: [`services/api`](../../services/api) (Go `/v1` API) and [`services/workers`](../../services/workers)
(scanner, scraper, domain verifier, QR, retention, DSAR, webhooks). Shared libraries:
[`pkg/ssrfguard`](../../pkg/ssrfguard), [`pkg/argon2id`](../../pkg/argon2id), [`pkg/idgen`](../../pkg/idgen),
[`pkg/scan`](../../pkg/scan), [`pkg/slug`](../../pkg/slug), [`pkg/urlpolicy`](../../pkg/urlpolicy).
OpenAPI 3.1: [`services/api/openapi.yaml`](../../services/api/openapi.yaml).

## 1. Auth, RBAC, tenant isolation

- **Email/password** hashed with Argon2id PHC (`m=65536 KiB, t=3, p=2`) — same parameters as link passwords.
- **JWT** HS256, user id only (no email in the token — PII minimization).
- **API keys** `lk_live_…`, SHA-256 at rest, prefix for UI, plaintext shown once, rotatable via revoke+create.
- **TOTP MFA**: secret envelope-encrypted (AES-GCM) in `users.mfa_totp_secret`; login requires the code when enabled.
- **OIDC**: `/v1/auth/oidc/start` redirects when `OIDC_ISSUER` is set; token exchange is environment-specific
  (501 until JWKS is wired for the chosen IdP).
- **RBAC**: viewer < member < admin < owner, loaded from `memberships` (or implied `admin` for API keys).
  Every mutating query is workspace-scoped (`workspace_id` in the WHERE clause).
- **Residency**: `home_region` (`us`/`eu`) is chosen at register and stored on the workspace.

## 2. Link creation path (matches Phase 1 sequence)

1. Validate destination (`http`/`https` only, length cap, blocked IP literals — `pkg/urlpolicy`).
2. **Fast sync scan < 20 ms**: heuristics + local Safe Browsing hash-prefix DB. Malicious → **422**, no row.
3. Allocate a 7-char code via **HiLo + Feistel** (`pkg/idgen` + `hilo_counters`), or a vanity alias
   (reserved-word table + profanity + charset).
4. Insert; uniqueness of `(domain_id, code)` is the database’s job (RPO=0, no cross-region dupes).
5. Async: scanner (URLHaus + heuristics + GSB), scraper, QR, webhooks (`link.created`).

`javascript:` / `data:` / loopback / metadata IPs are rejected at the API boundary **and** by the DB CHECK.

## 3. Workers

| Worker | Trigger | Notes |
|--------|---------|--------|
| **scanner** | pending/suspicious, plus 24h re-scan of clean | Pluggable `scan.Scanner`; URLHaus fetched with the SSRF client |
| **scraper** | missing/stale `link_metadata` | SSRF-pinned client, 5 redirects, 2 MB, 5 s; robots.txt; OG/Twitter/JSON-LD. Password links skipped (no destination leak) |
| **verifier** | `domains.next_check_at` | TXT + CNAME, exponential backoff, pending → verified → active / failed |
| **qr** | `qr_codes.object_key` empty | PNG + SVG, EC=H when logo; content includes `?s=qr` source marker |
| **retention** | periodic | Reads `retention_policies`, purges deleted accounts, reports TTL coverage |
| **dsar** | open `data_subject_requests` | erasure / restriction / GPC / access ticket completion |
| **webhooks** | `webhook_deliveries` due | HMAC-SHA256, SSRF-guarded POST, retry with backoff |

`WORKER_ROLE` selects one or `all`.

## 4. TLS ask

`GET /internal/tls/ask?domain=` + `X-Ask-Token` returns 200 only for `state=active`. Pending custom domains
are 404 so ACME cannot be abused (ADR-006).

## 5. Proven against CockroachDB (this phase)

| Call | Result |
|------|--------|
| `POST /v1/auth/register` | 201 JWT + workspace |
| `POST /v1/links` `https://example.com/hello` | 201 code `NoRuDiB` (Feistel, non-sequential) |
| `POST /v1/links` `javascript:alert(1)` | **422** problem+json |
| `POST /v1/domains` | 201 pending + TXT/CNAME instructions |
| `GET /internal/tls/ask?domain=lnk.linkspire.io` | **200** |
| `GET /internal/tls/ask?domain=go.ada.test` (pending) | **404** |
| `POST /v1/privacy/requests` access/GDPR | **202**, 30-day SLA |
| `POST /v1/api-keys` | **201**, `lk_live_…` shown once |

SSRF unit tests cover metadata IP, IPv4-mapped, dword/octal encodings, rebinding resolver, and scheme deny.

## Assumptions and Open Questions

- **Assumptions:** OIDC JWKS exchange is deferred to the chosen IdP; QR assets land on local disk in this
  phase (S3 is a drop-in `ObjectStore` in Phase 6); ClickHouse analytics queries return an empty page when
  CH is not configured rather than failing the 99.9% API SLO.
- **Open questions:** (1) WebAuthn ceremony endpoints (table exists; API in a follow-up). (2) Admin review
  UI for abuse reports (queue is `audit_log` + `scan_verdict`). (3) Confirm URLHaus CSV URL for production.
