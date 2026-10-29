<!-- docs/architecture/04-threat-model.md -->
# Phase 1.4 — Threat Model (STRIDE)

Scope: the three highest-risk surfaces — the **redirect hot path**, the **metadata scraper**, and
**custom domains**. STRIDE = Spoofing, Tampering, Repudiation, Information disclosure, Denial of service,
Elevation of privilege. Each threat lists the mitigation to be implemented in later phases.

---

## 1. Redirect hot path

**Assets:** link→destination mappings, link state (active/flagged/expired), password cookies, click-event
integrity, and *the privacy promise* (no raw IP/UA retained).

| # | STRIDE | Threat | Mitigation |
|---|--------|--------|-----------|
| R1 | **S** | Forged password cookie to bypass a protected link | Cookie is signed (HMAC/JWT) with a rotating key, `HttpOnly; Secure; SameSite=Strict`, scoped to one link id, short TTL; server-side key in Vault |
| R2 | **S** | Spoofed crawler user-agent to pull preview of a private/password link | Preview served **only** when overrides exist and link is not password-protected; UA is a hint, never an authz boundary; password links excluded from previews entirely |
| R3 | **T** | Tampering with `Location` / open-redirect injection via crafted code or host header | Destinations are validated & normalized at creation (reject `data:`/`javascript:`); redirect emits only the stored, canonicalized URL; host header validated against known domains |
| R4 | **T** | Cache poisoning (inject wrong destination into L1/Redis) | Only the service writes caches, keyed by `(domain_id,code)` from authenticated DB rows; values are integrity-checked structs; no user-controlled cache keys; TTL + jitter |
| R5 | **R** | User denies they created a malicious link | `audit_log` records create/edit with actor + hashed API key id + timestamp (no PII); immutable append |
| R6 | **I** | **Privacy breach — raw IP/UA leaking into logs or events** | Redirect path derives country/device/browser/os in memory then **discards the IP**; log-redaction middleware strips PII; click event schema has **no** IP/UA fields; unique-visitor = daily-rotating salted hash, salt destroyed in 24–48 h |
| R7 | **I** | Enumeration/scraping of codes to harvest destinations | Non-enumerable Feistel codes (ADR-003); **RedisBloom** rejects non-existent codes; **negative caching**; per-fingerprint rate limits; 404s are constant-time-ish and uninformative |
| R8 | **D** | Volumetric DDoS / cache-stampede on viral or non-existent codes | Edge WAF + anycast; **singleflight** coalesces misses; jittered TTLs; Bloom + negative cache absorb bogus lookups; per-key rate limits; autoscaling N+2 |
| R9 | **D** | Analytics backpressure stalls redirects | Click emission is **fire-and-forget** with a bounded buffer; on Kafka loss, drop analytics, **never** block the 3xx |
| R10 | **E** | A flagged-malicious link still redirects | State re-evaluated on every hit (302 default, `no-store`); flagged → warning interstitial, not a redirect; disabled/expired → 410 |

**Residual risk:** country-level geo in analytics is still derived from IP in memory; we mitigate by never
persisting the IP and truncating any IP that must touch an abuse log (IPv4 /24, IPv6 /48).

---

## 2. Metadata scraper (SSRF is the dominant risk)

**Assets:** internal network, cloud metadata endpoint (`169.254.169.254`), the scraper's egress identity,
and the correctness/safety of fetched preview data.

| # | STRIDE | Threat | Mitigation |
|---|--------|--------|-----------|
| S1 | **S** | DNS rebinding: hostname resolves public first, then flips to internal IP | **Resolve DNS first, pin the resolved IP**, and connect to that exact IP; re-validate on **every** redirect hop; reject if resolution changes to a blocked range |
| S2 | **T** | Attacker-controlled redirect chain pivots to internal service | Cap redirects at **5**; re-run the SSRF allowlist/denylist check at **each** hop; forbid cross-scheme downgrade to `file:`/`gopher:` etc. |
| S3 | **I** | **SSRF to cloud metadata / internal services** (the headline risk) | Block private, loopback, link-local, **`169.254.0.0/16`**, ULA, and **IPv6-mapped IPv4** ranges; deny non-HTTP(S) schemes; scraper runs in an **isolated, egress-restricted network segment** with no IAM role and metadata endpoint blocked at the host/network level |
| S4 | **I** | Oversized/zip-bomb or slow-loris response exhausts the worker | Cap response size **2 MB**, total time **5 s**, connect/read timeouts; stream with a hard byte ceiling; reject non-text content types for parsing |
| S5 | **D** | Scraper used as a DoS amplifier against third parties | Per-destination fetch rate limits; respect `robots.txt` where applicable; dedup + refresh-TTL cache so one URL isn't re-fetched in a storm |
| S6 | **R** | Dispute over what preview content was fetched/when | Store fetch timestamp, final resolved URL, HTTP status, and content hash in `link_metadata`/`scan_results` |
| S7 | **E** | Parser exploit (malicious HTML/JSON-LD) escalates in the worker | Memory-safe Go parsers, no `eval`, no headless browser by default; run as non-root distroless with seccomp; drop all unneeded capabilities |

**Design note.** The SSRF guard is a single shared library (`pkg/ssrfguard`) used by the scraper **and** the
scanner, with a dedicated test suite (Phase: Testing) covering rebinding, redirect pivots, IPv6-mapped,
decimal/octal IP encodings, and metadata-endpoint attempts.

---

## 3. Custom domains

**Assets:** domain ownership integrity, TLS private keys, the ACME account, SNI routing correctness, and
tenant isolation across domains.

| # | STRIDE | Threat | Mitigation |
|---|--------|--------|-----------|
| D1 | **S** | Claiming a domain the attacker doesn't own | Two-factor proof: **TXT token** ownership record **and** CNAME pointing at us; verifier requires both before `verified`; tokens are high-entropy and single-use per domain |
| D2 | **S** | Dangling-CNAME / subdomain takeover of a once-verified domain | Periodic re-verification; on CNAME removal → state `revoked`, stop serving + stop cert renewal; `ask` endpoint denies de-authorized hosts |
| D3 | **T** | Tampering with the TLS `ask` decision to mint unauthorized certs | `ask` endpoint authorizes **only `verified`+`active`** domains via authenticated internal call; allow/deny result cached briefly; mutual-auth between edge and API |
| D4 | **I** | TLS private key disclosure | Keys stored in the shared cert store encrypted at rest; least-privilege access; never logged; rotation on renewal |
| D5 | **I** | Cross-tenant data exposure via SNI/host confusion | Host→(domain_id, tenant) mapping is authoritative from DB; all queries filtered by `domain_id` + tenant (RLS/query-layer isolation); no wildcard trust |
| D6 | **D** | ACME rate-limit exhaustion (self-inflicted or attacker-driven) | Authorize only verified domains; backoff + jitter; issuance queue with per-account caps; monitor Let's Encrypt limits; fall back to ZeroSSL |
| D7 | **D** | Verifier hammered by mass fake domain adds | Rate-limit domain creation per workspace; exponential backoff with cap; cap pending domains per tenant |
| D8 | **E** | A customer domain used to phish under our TLS/infra | URL scanning applies to links on custom domains too; abuse-report endpoint + admin review queue; revoke domain + certs on confirmed abuse |
| D9 | **R** | Dispute over verification/issuance history | `audit_log` + domain state transitions recorded with timestamps and the DNS evidence observed |

**Apex-domain limitation (documented):** apex (`example.com`) cannot CNAME per RFC 1034; we rely on
ALIAS/ANAME at providers that support it, else an A-record fallback to our anycast IPs — which the customer
must update if our IPs change. This is called out in the domain setup wizard (Phase 5).

---

## Cross-cutting controls (apply to all surfaces)

- **AuthN/AuthZ:** OIDC/OAuth 2.1 + email/password with MFA (TOTP/WebAuthn); scoped, hashed, rotatable API
  keys; RBAC per workspace (owner/admin/member/viewer); tenant isolation at the query layer + Postgres RLS
  where supported.
- **Secrets:** Vault/Secrets Manager only; never in committed env files.
- **Supply chain:** dependency scanning, SBOM, signed container images, least-privilege service accounts,
  distroless non-root images.
- **Headers:** CSP, HSTS, X-Content-Type-Options, Referrer-Policy; input validation on every boundary;
  open-redirect safety on the dashboard itself (allowlist of internal return URLs).

## Assumptions and Open Questions

- **Assumptions:** the scraper node pool can be network-isolated with the cloud metadata endpoint blocked;
  edge↔API `ask` channel can be mutually authenticated; RLS is available in CockroachDB or enforced
  equivalently at the repository layer.
- **Open questions:** (1) Do we ever need a headless-browser renderer for JS-heavy previews? If so it needs
  its own hardened, egress-locked sandbox and a separate threat review. (2) Should confirmed-malicious
  custom domains be reported to upstream registrars/Safe Browsing automatically? (3) Acceptable abuse-log
  retention window for truncated IPs (assumed 90 days).
