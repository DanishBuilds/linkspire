<!-- README.md -->
# Linkspire — Privacy-First URL Shortener Platform

> Working name: **Linkspire**. Assumption stated here because the brief used `[ProductName]`.
> Public CNAME target: `cname.linkspire.io`. Edge domain: `lnk.spr` (example).

Linkspire is an internet-scale, privacy-by-design link platform targeting **100M redirects/day**
average, **50k req/s** burst, redirect **p99 < 30 ms** at origin, and **99.99%** redirect
availability with **RPO = 0** for committed link data.

This repository is delivered in phases. **Phase 1 (Architecture)** is complete in `docs/architecture/`.
Subsequent phases (data layer, redirect service, management API/workers, dashboard, infra, compliance)
build on these decisions.

## Phase 1 deliverables

| Doc | Contents |
|-----|----------|
| [`docs/architecture/01-capacity-estimation.md`](docs/architecture/01-capacity-estimation.md) | QPS, 5-year storage, cache/Bloom memory, Redis/Postgres/ClickHouse node counts — with explicit math |
| [`docs/architecture/02-architecture-and-flows.md`](docs/architecture/02-architecture-and-flows.md) | High-level diagram + request-flow diagrams (redirect, create, password unlock, domain onboarding, DSAR erasure), degradation matrix, SLOs |
| [`docs/architecture/03-adr.md`](docs/architecture/03-adr.md) | ADRs: language, distributed SQL engine, ID generation, redirect status default, analytics store, TLS strategy |
| [`docs/architecture/04-threat-model.md`](docs/architecture/04-threat-model.md) | STRIDE threat models for redirect path, scraper, and custom domains |

## Planned repository tree (target end-state across all phases)

```text
linkspire/
├── README.md
├── docs/
│   ├── architecture/            # Phase 1 (this phase)
│   ├── compliance/              # Phase 7: RoPA, DPA, privacy policy, retention schedule
│   └── runbooks/                # Phase 7: cache failure, DB failover, cert outage, mass-phishing, DSAR
├── services/
│   ├── redirect/                # Phase 3: Go hot-path service (L1 LRU, Redis, singleflight, Bloom)
│   ├── api/                     # Phase 4: Go management API (/v1, OIDC, RBAC, idempotency)
│   └── workers/                 # Phase 4: scanner, scraper (SSRF-guarded), domain-verifier, qr, retention, dsar
├── pkg/                         # Shared Go libs: codec (Feistel/Base62), ssrfguard, ratelimit, otel, config
├── proto/                       # Click-event schema (protobuf), webhook payloads
├── db/
│   ├── migrations/              # golang-migrate SQL (CockroachDB)
│   └── clickhouse/              # ClickHouse DDL + TTLs
├── web/                         # Phase 5: dashboard (Next.js, WCAG 2.2 AA)
├── deploy/
│   ├── docker/                  # Phase 6: distroless, non-root Dockerfiles
│   ├── helm/                    # Phase 6: charts + HPA + PodDisruptionBudgets
│   └── terraform/               # Phase 6: cloud modules
├── test/
│   ├── load/                    # Phase 6: k6 scripts + results
│   └── security/                # SSRF suite, rate-limit bypass, authz/tenant isolation, decoder fuzzing
└── .github/workflows/           # Phase 6: CI/CD, SAST, image scan, SBOM, canary
```

## Core principles (binding constraints)

1. **No synchronous third-party network call on the redirect hot path. Ever.**
2. **Never log or persist raw IPs, full user agents, or passwords.** Country/device/browser/OS are
   derived in memory; the IP is discarded immediately.
3. **Privacy wins ties.** When analytics accuracy conflicts with privacy, choose privacy, document the
   tradeoff, and offer an opt-in alternative.
4. **Boring, proven technology** unless a measured target demands otherwise.
