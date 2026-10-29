<!-- docs/architecture/06-redirect-service.md -->
# Phase 3 — Redirect Service (Go)

Runnable code under [`services/redirect/`](../../services/redirect) and shared code generator
under [`pkg/codec/`](../../pkg/codec). Module: `github.com/linkspire/linkspire` (Go 1.22+).

## 1. Package layout

```text
pkg/codec/                     # shared: Base62 + keyed Feistel permutation (ADR-003)
  base62.go  feistel.go  generator.go  codec_test.go   (+ fuzz + bench)
services/redirect/
  main.go                      # DI wiring, dual HTTP servers, graceful shutdown
  internal/
    config/      config.go     # typed env config, validated at startup
    model/       link.go       # hot-path link projection (+ EffectiveStatus)
    cache/       l1.go         # in-process expirable LRU (tier 1)
                 bloom.go      # existence filter: LocalBloom + key helpers
                 redis.go      # L2: records, negative cache, RedisBloom, visitor salt
                 kv.go         # counters/locks for brute-force protection
    resolver/    resolver.go   # L1→Bloom→Redis→PG, singleflight, jittered backfill
                 resolver_test.go
    privacy/     privacy.go    # IP truncation, visitor hash, UA classify, geo, referrer
                 salt.go       # rotating daily salt provider
    events/      emitter.go    # bounded-buffer Kafka emitter (fire-and-forget) + Nop
    observability/ metrics.go  # Prometheus collectors
                 tracing.go    # OpenTelemetry OTLP setup (no-op when disabled)
    handler/     handler.go    # redirect state machine + security headers
                 password.go   # Argon2id verify, signed cookie, CSRF, lockout
                 limiter.go    # Redis-backed attempt limiter
                 host.go       # Host header → domain_id (cached)
                 health.go     # liveness / readiness
                 interstitial.go + templates/*.html
                 handler_test.go
```

## 2. Hot-path guarantees (as implemented)

- **Lookup order** `L1 → Bloom → Redis(+negative) → Postgres → backfill`, in `resolver.Resolve`.
  Any tier may be disabled; **cache errors fail open** to the next tier so redirects keep serving.
- **Stampede protection:** `golang.org/x/sync/singleflight` coalesces concurrent misses for the
  same code into **one** DB read (unit-tested: 50 concurrent misses → 1 repo call).
- **Jittered TTLs:** record and negative backfills add ±jitter to prevent synchronized expiry.
- **Bloom correctness note:** the Bloom is only trusted when it contains *all* codes, so it defaults
  **off** and is wired to the shared **RedisBloom** in production (maintained by the creation
  pipeline). An empty in-process filter would wrongly reject valid links — guarded against in config.
- **No synchronous third-party call.** The only hot-path I/O is to our own Redis/Postgres.
- **Async, non-blocking analytics:** `events.Kafka` uses a bounded ring buffer; a full buffer
  **drops** the event (metric `...events_dropped_total`) and the redirect proceeds. Kafka down ⇒
  analytics lossy, redirects unaffected.
- **Status handling** (`model.EffectiveStatus` + handler switch): active → redirect;
  expired/click-limit → 410; disabled → 410; flagged-malicious → warning interstitial (no auto
  follow, no `Location`); password-protected → CSRF-protected prompt / signed-cookie bypass.
- **Status codes:** default 302 with `Cache-Control: no-store`; 301/308 emit `public, max-age=...`
  (ADR-004).

## 3. Privacy (as implemented)

- `privacy.go` derives **country/region (stub GeoIP), device/browser/os (UA parsed in memory then
  discarded), referrer domain only**, and a **daily-salted visitor hash**. There is **no struct
  field, log line, or event field** for a raw IP or full user-agent.
- IP truncation helpers: IPv4 `/24`, IPv6 `/48`, used only to derive the anonymized fingerprint
  for rate limiting.
- The rotating salt is fetched from Redis and cached in memory; the hot path never blocks on it.

## 4. Security (as implemented)

- Password unlock: **Argon2id** PHC verify (`golang.org/x/crypto/argon2`), constant-time compare,
  never logs plaintext; short-lived **HMAC-signed, HttpOnly, Secure, SameSite=Strict** cookie scoped
  to the link path; **CSRF double-submit** token; **per-link+client lockout** with progressive delay.
- Response-splitting defense on `Location`; auto-escaping `html/template` for interstitials with a
  strict CSP; baseline headers (`X-Content-Type-Options`, `X-Frame-Options`) on every response.

## 5. Operations

- Typed config with `Validate()` (refuses to boot in prod with the default cookie key, etc.).
- Dual servers: public redirect (`:8080`) and private admin (`:9090`: `/metrics`, `/healthz`,
  `/readyz`). Readiness only gates on **hard** deps (Postgres); Redis/Kafka degrade gracefully.
- Graceful shutdown drains the emitter and flushes traces within `SHUTDOWN_TIMEOUT`.
- OpenTelemetry tracing (OTLP/HTTP, parent-based 1% sampling) — no-op when disabled, so hot-path
  span cost is negligible.

## 6. Tests & benchmarks (executed)

| Check | Result |
|-------|--------|
| `go build ./...` | pass |
| `go vet ./...` | clean |
| codec unit + fuzz seed + no-collision + non-sequential | pass |
| resolver: L1/Redis/PG tiers, negative cache, bloom reject, **singleflight coalescing (50→1)** | pass |
| handler: 302/301 cache policy, 404, warning, 410 expired, password prompt+CSRF, 503 degradation | pass |
| **End-to-end vs real CockroachDB** | `GET /aB3xY7z` → `302 Location: https://example.com/landing`, `no-store`; repeat served from L1 (PG hit stayed 1) |
| `BenchmarkResolveL1Hit` | **77 ns/op, 1 alloc** (hot path) |
| `BenchmarkEncode` (Feistel code gen) | 4.4 µs/op (creation path, not hot path) |

The L1-hit resolve at ~77 ns is ~4 orders of magnitude under the p99<5 ms cache-hit budget; HTTP
framing dominates real latency and remains far inside the 30 ms p99 origin target.

## Assumptions and Open Questions

- **Assumptions:** GeoIP is a stub interface (MaxMind GeoLite2 wired in Phase 6 via `GEO_DB_PATH`);
  JSON is used for Redis values and Kafka payloads for clarity (msgpack/protobuf is a drop-in
  optimization); RedisBloom is reserved/populated out-of-band by the creation pipeline.
- **Open questions:** (1) Should flagged-link "continue anyway" clicks be emitted as a distinct
  analytics source? (2) Do we want cross-pod singleflight (the optional `sf:` Redis lock) enabled, or
  is per-pod coalescing sufficient at our pod count? (3) Confirm Argon2id params (m=64MB,t=3,p=2) meet
  your latency budget for the unlock endpoint under load.
