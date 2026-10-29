<!-- docs/architecture/02-architecture-and-flows.md -->
# Phase 1.2 — High-Level Architecture & Request Flows

## 1. System context (high level)

```mermaid
flowchart TB
    subgraph Clients
        U[End users / browsers]
        C[Crawlers: Slackbot, Twitterbot, ...]
        D[Dashboard users]
        API[API consumers]
    end

    subgraph Edge["Edge / CDN (Cloudflare-class) + WAF"]
        WAF[WAF + bot rules + Turnstile]
        TLS[On-demand TLS<br/>Caddy ask-endpoint for custom domains]
    end

    subgraph RegionA["Region (active) — repeated per region"]
        direction TB
        RS[Redirect Service - Go<br/>L1 LRU + singleflight]
        MA[Management API - Go<br/>OIDC/RBAC/idempotency]
        subgraph Workers
            SCAN[Scanner pipeline]
            SCRAPE[Scraper - egress-restricted pool]
            VER[Domain verifier]
            QR[QR generator]
            RET[Retention job]
            DSAR[DSAR processor]
        end
        RED[(Redis Cluster + RedisBloom<br/>L2 cache / negative cache)]
        PG[(CockroachDB<br/>multi-region, RF=3)]
        KAF[[Redpanda<br/>click events + jobs]]
        CH[(ClickHouse<br/>analytics, TTL)]
        S3[(S3-compatible<br/>QR assets)]
        VAULT[[Secrets Vault]]
    end

    U --> WAF --> RS
    C --> WAF --> RS
    D --> WAF --> MA
    API --> WAF --> MA
    TLS -.authorize verified domains.-> MA

    RS -->|miss| RED
    RS -->|miss| PG
    RS -.fire-and-forget.-> KAF
    MA --> PG
    MA --> RED
    MA --> KAF
    MA --> S3

    KAF --> SCAN
    KAF --> SCRAPE
    KAF --> QR
    KAF --> CH
    VER --> PG
    SCAN --> PG
    SCRAPE --> PG
    QR --> S3
    RET --> PG
    RET --> CH
    DSAR --> PG
    DSAR --> CH
    DSAR --> S3
    DSAR --> RED

    MA --> VAULT
    RS --> VAULT
```

**Tiering principles**

- **Hot path = Redirect Service only.** It talks to L1 (in-proc), Redis, and a Postgres read replica. It
  **never** makes a synchronous third-party call. Click events are fire-and-forget to Redpanda with a
  bounded local buffer and backpressure (drop analytics, never redirects).
- **Everything expensive is async:** scanning, scraping, QR, metadata, analytics rollups — all consume
  from Redpanda.
- **Each region is a full active stack.** CockroachDB spans regions for global code uniqueness and RPO=0;
  Redis/ClickHouse/Redpanda are per-region. Edge steers users to the nearest healthy region.

## 2. Redirect hot path

```mermaid
flowchart TD
    A[GET /:code on domain] --> B{L1 in-proc LRU hit?}
    B -- yes --> ST[Resolve link state]
    B -- no --> BL{RedisBloom: code maybe-exists?}
    BL -- definitely-no --> NEG[Return 404<br/>set negative cache + metric]
    BL -- maybe --> R{Redis GET hit?}
    R -- yes --> FILL1[Backfill L1] --> ST
    R -- no --> SF[singleflight: coalesce<br/>concurrent misses per code]
    SF --> PG{Postgres read-replica hit?}
    PG -- no --> NEG2[404 + negative cache jittered TTL]
    PG -- yes --> FILL2[Backfill Redis jittered TTL + L1] --> ST

    ST --> S1{state}
    S1 -- active --> EMIT[enqueue click event - non-blocking]
    S1 -- expired/disabled --> GONE[410 Gone / custom 404]
    S1 -- flagged-malicious --> WARN[Serve warning interstitial]
    S1 -- password --> PW[Serve password interstitial<br/>check signed cookie first]

    EMIT --> RT{redirect_type}
    RT -- 301/308 --> RPERM[Permanent redirect<br/>Cache-Control allows edge/browser cache]
    RT -- 302/307 --> RTMP[Temporary redirect<br/>no-store so each hit reaches origin]
```

**Hot-path guarantees**

- Lookup order **L1 → Bloom → Redis → Postgres replica → backfill both caches** (§ matches brief).
- **Stampede protection:** `singleflight` coalesces concurrent misses for the same code into one DB read;
  all cache TTLs are jittered to avoid synchronized expiry.
- **Negative caching + Bloom** absorb enumeration attacks before they reach Redis/Postgres.
- **Click emission is non-blocking:** bounded ring buffer → async producer. If Redpanda is unavailable the
  buffer drains to a local WAL then drops oldest; **the redirect is issued regardless.**
- **Status default = 302** (analytics-friendly); 301/307/308 configurable per link (see ADR-004).

## 3. Link creation

```mermaid
sequenceDiagram
    participant Cli as Client (API/UI)
    participant API as Management API (Go)
    participant Scan as Fast sync check (<20ms)
    participant PG as CockroachDB
    participant K as Redpanda
    participant W as Async workers

    Cli->>API: POST /v1/links (Idempotency-Key, dest, opts)
    API->>API: AuthN (OIDC/API key), RBAC, rate-limit budget
    API->>API: Validate URL; reject data:/javascript:; normalize
    API->>Scan: local Safe-Browsing hash DB + heuristics (<20ms)
    alt immediately malicious
        API-->>Cli: 422 rejected (problem+json)
    else clean/suspicious-pending
        API->>PG: INSERT link (code via Feistel; unique(domain_id,code))
        Note over PG: Global unique index → no cross-region dup codes (RPO=0)
        PG-->>API: committed
        API->>K: publish link.created
        API-->>Cli: 201 Created (short URL, QR pending)
        K->>W: deep scan, QR gen, metadata scrape (async)
        W->>PG: update scan_verdict / qr_codes / link_metadata
        W->>K: webhook link.created / link.flagged (if verdict flips)
    end
```

- Code is generated by a keyed Feistel permutation over the Base62 space (ADR-003) — no DB round-trip to
  pick an ID, non-enumerable by construction.
- **Creation never waits on slow scanners.** A sub-20 ms local check gates obvious abuse; deep scanning is
  async and flips state + fires `link.flagged` + notifies owner if needed.
- **Idempotency-Key** makes retries safe (dedup table keyed by `(workspace, key)`).

## 4. Password unlock

```mermaid
sequenceDiagram
    participant B as Browser
    participant RS as Redirect Service
    participant RL as Redis (rate-limit + lockout)
    participant PG as CockroachDB

    B->>RS: GET /:code  (password-protected link)
    RS->>RS: No valid signed cookie → serve CSRF-protected interstitial
    B->>RS: POST password (+ CSRF token)
    RS->>RL: check per-link & per-client attempt budget
    alt locked out
        RS-->>B: 429 Retry-After (progressive delay)
    else allowed
        RS->>PG: fetch Argon2id hash (cached in Redis, not the plaintext)
        RS->>RS: Argon2id verify (never log plaintext)
        alt match
            RS->>RS: issue short-lived signed cookie<br/>HttpOnly, Secure, SameSite=Strict, scoped to link
            RS-->>B: 302 → destination
        else mismatch
            RS->>RL: increment attempts, apply progressive delay
            RS-->>B: 401 interstitial with error
        end
    end
```

- Password-protected links are **excluded from metadata previews** (no OG/destination leakage to crawlers).
- Cookie is scoped to the single link; verification is Argon2id with documented, tuned parameters.

## 5. Custom-domain onboarding (CNAME + on-demand TLS)

```mermaid
sequenceDiagram
    participant U as User
    participant API as Management API
    participant V as Domain Verifier (worker)
    participant DNS as Public DNS
    participant Edge as Edge TLS (Caddy on-demand)
    participant ACME as Let's Encrypt/ZeroSSL

    U->>API: POST /v1/domains {go.acme.com}
    API-->>U: CNAME → cname.linkspire.io  +  TXT _linkspire-verify=<token>
    Note over U,DNS: User adds records at their DNS provider
    loop exponential backoff
        V->>DNS: resolve TXT + CNAME
        alt both present & correct
            V->>API: state pending → verified → active
        else
            V->>V: backoff, retry (cap), else state=failed
        end
    end
    Note over Edge,ACME: First HTTPS hit to go.acme.com
    Edge->>API: TLS "ask": is go.acme.com authorized?
    API-->>Edge: yes (verified+active) / no
    alt authorized
        Edge->>ACME: ACME order (rate-limit aware), SNI cert
        ACME-->>Edge: certificate (auto-renew)
        Edge-->>U: HTTPS served
    else
        Edge-->>U: refuse TLS (prevents cert abuse)
    end
```

- Apex domains: ALIAS/ANAME where the provider supports it; A-record fallback documented as a limitation.
- Per-domain settings: root-path default redirect, custom 404, HSTS toggle.
- The **ask endpoint only authorizes verified+active domains**, bounding ACME issuance and abuse.

## 6. DSAR erasure (right to be forgotten)

```mermaid
sequenceDiagram
    participant U as Data subject
    participant API as Management API
    participant DQ as data_subject_requests (SLA timer)
    participant DSAR as DSAR processor (worker)
    participant PG as CockroachDB
    participant CH as ClickHouse
    participant RED as Redis
    participant S3 as Object storage
    participant AUD as audit_log

    U->>API: POST /v1/privacy/requests {type: erasure}
    API->>DQ: create request (SLA: 30d GDPR / 45d CCPA), verify identity
    API-->>U: 202 Accepted (request id, status URL)
    DQ->>DSAR: dispatch
    DSAR->>PG: delete/anonymize user, memberships, links, link_passwords, qr meta
    DSAR->>S3: delete QR assets for owned links
    DSAR->>RED: evict cached codes + negative entries + bloom note
    DSAR->>CH: delete/anonymize analytics rows (lightweight delete / partition rewrite)
    DSAR->>AUD: write tamper-evident audit entry (who/what/when, no PII)
    DSAR->>DQ: mark completed + machine-readable report
    DSAR-->>U: completion notice (export link if access request)
```

- Erasure **cascades across every store**: Postgres, caches (L1 handled via TTL + explicit bust), object
  storage, analytics, and search/index if present.
- The `audit_log` records the action **without** storing the erased PII.

## 7. Degradation matrix

| Failure | Behavior | Guarantee |
|---------|----------|-----------|
| **Redis down** | Serve from L1; misses go straight to Postgres read replicas; tighten rate limits; disable negative-cache writes | Redirects keep flowing (higher p99); writes unaffected |
| **Kafka/Redpanda down** | Click events buffer in bounded local ring → local WAL; drop oldest when full | **Redirects never blocked**; analytics lossy during outage (documented) |
| **Postgres replica down** | Route reads to another replica; fall back to primary with circuit breaker | Reads continue; writes unaffected |
| **Postgres primary/region down** | Automated CockroachDB region failover; standby promoted | RTO < 5 min; RPO = 0 (consensus-committed) |
| **Scanner feed down** | Fast local DB still applies; deep scan queued; fail safe = suspicious until verified for risky links | Creation not blocked |
| **Scraper egress down** | Preview falls back to user overrides or no card | Redirect unaffected |
| **ACME/cert outage** | Serve existing (unexpired) certs; queue issuance; alert | Existing custom domains keep working |
| **Edge/CDN region down** | Anycast/health-checked DNS steers to next region | 99.99% budget preserved |

## 8. SLOs & error budgets (redirect path)

| SLI | Target | Window |
|-----|--------|--------|
| Redirect availability (non-5xx / total) | 99.99% | 30-day rolling |
| Redirect latency (origin, cache hit) | p99 < 5 ms | 5-min + 1-h |
| Redirect latency (origin, overall) | p50 < 10 ms, p99 < 30 ms | 5-min + 1-h |
| Click-event delivery (best-effort) | ≥ 99.5% delivered | 1-day |

- **Multi-window, multi-burn-rate alerts** (e.g. fast: 2%/1h at 14.4x burn; slow: 5%/6h) on the
  availability and latency SLIs, wired to OpenTelemetry → Prometheus → Alertmanager.
- Error budget policy: when budget is exhausted, freeze risky deploys to the redirect path until recovered.

## Assumptions and Open Questions

- **Assumptions:** Cloudflare-class edge provides WAF + Turnstile + anycast; CockroachDB multi-region is the
  authority for global code uniqueness; each region runs independent Redis/ClickHouse/Redpanda; the edge can
  perform on-demand TLS with an ask endpoint (Caddy or equivalent at the edge/ingress).
- **Open questions:** (1) Is edge caching of 301/308 acceptable to product given it suppresses per-hit
  analytics for those links? (2) For CCPA erasure of analytics, is anonymization (drop join keys) acceptable
  vs hard delete, to preserve aggregate counts? (3) Do we need active-active *writes* per region, or
  active-passive with a single write home region per tenant (residency)?
