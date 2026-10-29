<!-- docs/architecture/03-adr.md -->
# Phase 1.3 — Architecture Decision Records

Format: Context → Options → Decision → Consequences. Status: Accepted (Phase 1).

---

## ADR-001 — Language & stack split

**Context.** We need p99 stability under 50k req/s on the redirect path and a productive stack for the
management API, workers, and dashboard. The brief allows Go for the hot path and Go *or* Node (TS) for the
API/workers.

**Options.**
- (A) **Go everywhere** for backend (redirect + API + workers); dashboard in TS.
- (B) Go for redirect, **Node/TS (Fastify/NestJS)** for API/workers.
- (C) Rust for redirect, Node for the rest.

**Decision.** **(A) Go for all backend services; Next.js/React (TypeScript) for the dashboard only.**

**Rationale.**
- The redirect path demands low GC pause variance and low memory per connection — Go delivers predictable
  p99 with goroutine-per-request and a mature net/http stack, without Rust's higher implementation cost.
- Choosing **one backend language** eliminates a polyglot operational surface: a single toolchain, one set
  of base images, shared libraries for the **short-code codec (Feistel/Base62)**, **SSRF guard**,
  **rate-limiter**, config, and OpenTelemetry wiring. The SSRF guard in particular is security-critical and
  is far safer maintained once, in one language, than duplicated.
- Node/TS was considered for API velocity and ecosystem, but the shared security/codec libraries and single
  deploy pipeline outweigh it. TypeScript is still used where it is strongest — the browser dashboard.
- Rust (C) was rejected for the hot path on cost/complexity grounds; Go already meets the p99 target with
  headroom (see capacity §6.1).

**Consequences.** Slightly more boilerplate than NestJS for CRUD; mitigated with code generation from the
OpenAPI spec. One hiring profile for backend. Shared `pkg/` modules reused across services.

---

## ADR-002 — Distributed SQL engine

**Context.** Link data requires **RPO = 0**, **global uniqueness of `(domain_id, code)` across regions**
(no duplicate codes even under concurrent multi-region creation), horizontal scale to multi-TB, data
**residency pinning** (EU/US), and Postgres compatibility to reuse tooling.

**Options.**

| | **Citus** (sharded Postgres) | **CockroachDB** | **YugabyteDB** |
|---|---|---|---|
| Model | Coordinator + workers, sharded single-region primary | Native distributed, Raft per range, multi-region | Native distributed (YSQL on DocDB/Raft) |
| Global unique on non-distribution column | Hard — unique must include distribution key or be enforced per-shard | **Native** global unique secondary index via consensus | Native global unique index |
| Multi-region strong consistency | Weak (async replica regions) | **Serializable, region survival goals, REGIONAL BY ROW** | Serializable, tablespace/geo-partitioning |
| RPO=0 across region failure | Needs careful sync replica setup | **Yes (Raft quorum)** | Yes (Raft quorum) |
| Residency pinning | Manual | **Table/row locality built-in** | Tablespace geo-partition |
| PG compatibility | Highest (is Postgres) | High (PG wire, some gaps) | High (PG wire) |
| Operational maturity for this use | High | High | High (slightly younger) |

**Decision.** **CockroachDB.**

**Rationale.**
- The **make-or-break requirement is global uniqueness of short codes under concurrent multi-region
  writes with RPO=0.** CockroachDB enforces a **unique secondary index on `(domain_id, code)` via Raft
  consensus** — duplicate codes are impossible by the database, not by hopeful app-level coordination.
  Citus cannot cheaply enforce uniqueness on `code` unless `code` is the distribution key, which then
  fights our tenant-colocation needs; and Citus multi-region is asynchronous (RPO > 0 on region loss).
- **`REGIONAL BY ROW`** gives per-tenant **data residency pinning** (EU/US) in one table, directly serving
  the compliance requirement, while global tables (reserved words, blocklists) replicate everywhere.
- Serializable isolation removes a class of concurrency bugs for billing/quota logic.
- YugabyteDB is a strong equivalent; CockroachDB chosen for its more mature multi-region locality UX and
  operational tooling. This is a reversible choice behind the repository layer.

**Consequences.** Higher write latency than single-node Postgres (quorum cost) — irrelevant here because
writes are ~12/s avg / 500/s burst. Some Postgres features are unsupported; migrations use golang-migrate
with Cockroach-compatible DDL. We still front it with **PgBouncer (transaction mode)** for connection fan-in.

---

## ADR-003 — Short-code ID generation

**Context.** Codes must be collision-free, **non-sequential and non-enumerable** (not guessable/scrapable),
7 chars Base62 by default, generated without a hot-path DB round-trip or single coordinator SPOF, and must
support custom vanity aliases.

**Options.**
- (A) **DB sequence → Base62.** Sequential, trivially enumerable. Reject.
- (B) **Random 7-char Base62 with retry-on-collision.** Needs a uniqueness check (DB round-trip) and suffers
  birthday collisions as fill grows. Adds latency/complexity.
- (C) **Snowflake 64-bit → Base62.** 64 bits needs **11** Base62 chars, not 7. Also monotonic-ish → leaks
  ordering/time. Reject for *codes* (we still use Snowflake-style 64-bit IDs for internal entity PKs/events).
- (D) **Pre-allocated key ranges per node (HiLo) over the code integer space + keyed Feistel permutation,
  then Base62 encode.** No per-code DB round-trip, no central coordinator on the hot path, collision-free by
  construction, non-enumerable because the Feistel permutation scrambles sequential counters.

**Decision.** **(D).** Each code-generating node leases a block of integers from the `62^7` space
(`N = 3,521,614,606,208`) via a HiLo range allocator (one cheap DB round-trip per *block*, e.g. 10k ids).
A **keyed Feistel network** (balanced, ~4 rounds, key in Vault) is applied as a **format-preserving
permutation over `[0, N)`** using **cycle-walking** to stay within the Base62-7 domain. The permuted integer
is Base62-encoded to exactly 7 chars.

**Why this satisfies the requirements.**
- **Collision-free:** a bijection over `[0, N)` + unique block leases ⇒ no two inputs map to the same code.
- **Non-enumerable:** the Feistel permutation makes consecutive counter values produce uncorrelated codes;
  without the key you cannot predict the next/neighboring code — blunts scraping/enumeration.
- **No hot-path coordination:** blocks are leased rarely and cached per node; code emission is pure CPU.
- **Capacity:** `N = 3.52e12 ≫ 1.825e9` five-year demand (capacity §2) → length 7 is ample.
- **Vanity aliases** bypass the generator: validated against a **reserved-word blocklist + profanity
  filter**, then inserted with the same `unique(domain_id, code)` constraint (ADR-002 enforces global
  uniqueness). Reserved/blocked slugs live in a global (every-region) table.

**Consequences.** Feistel key is a long-lived secret; rotation requires versioned keys (a 1-byte key-epoch
prefix is reserved in the internal id, codes remain 7 chars). Base62 alphabet and round function are
documented and unit/fuzz-tested in `pkg/codec` (Phase 2/3).

---

## ADR-004 — Default redirect status code

**Context.** Status code choice trades **analytics completeness** against **browser/edge caching (latency,
origin load, SEO link equity)**.

| Code | Semantics | Caching | Analytics | SEO |
|------|-----------|---------|-----------|-----|
| 301 | Permanent | Browser/edge cache aggressively | **Misses most repeat hits** | Passes link equity |
| 308 | Permanent (method-preserving) | Same as 301 | Same gap | Same |
| 302 | Temporary | Not cached by default | **Each hit reaches origin → counted** | Does not pass equity |
| 307 | Temporary (method-preserving) | Not cached | Same as 302 | Same |

**Decision.** **Default = 302**, per-link configurable to 301/307/308.

**Rationale.** The platform's value is **per-hit analytics**; a permanent redirect would let browsers/CDNs
short-circuit repeat visits so clicks go uncounted. 302 keeps each hit arriving at the origin where it is
counted and where state (expiry, disable, flag) is re-evaluated — important because a link can become
malicious *after* shortening. We set `Cache-Control: no-store` on 302/307 responses to make this reliable.

**Consequences.** Higher origin QPS than a cache-everything design (sized for in capacity §1/§6). Users who
prioritize latency/SEO over analytics opt into 301/308; we document that those links will undercount and may
be edge-cached. Method-preserving 307/308 offered for non-GET flows.

---

## ADR-005 — Analytics store

**Context.** 100M events/day, high-cardinality aggregation (by link/country/device/browser/os/referrer),
TTL-based retention (raw 30 d, rollups 25 months), cheap storage, self-hostable, privacy-safe (no raw IP/UA).

**Options.** PostgreSQL/Citus (reuse) · **ClickHouse** · Apache Druid/Pinot · managed warehouse (BigQuery).

**Decision.** **ClickHouse.**

**Rationale.** Columnar storage + extreme compression (capacity §4.2: ~6× conservative) make 100M/day
cheap; `MergeTree` with **native TTL** cleanly enforces retention (`TTL event_time + INTERVAL 30 DAY`) and
**materialized views** maintain 25-month rollups incrementally. Aggregation queries over low-cardinality
privacy-safe dimensions are its sweet spot. Postgres/Citus would bloat and slow on this volume; Druid/Pinot
add operational weight we don't need at this scale; a managed warehouse conflicts with self-host/residency
and cost-per-event goals.

**Consequences.** A second datastore to operate (sized small, 2–3 nodes/region). Eventual consistency for
rollups via materialized views (acceptable for analytics). Deletes for DSAR use lightweight deletes /
partition rewrites — slower than Postgres but bounded and auditable.

---

## ADR-006 — Custom-domain TLS strategy

**Context.** We must terminate HTTPS for **arbitrary customer domains** (CNAME'd to us) without manual
per-domain certificate config, with auto-renewal, ACME rate-limit awareness, and authorization so we don't
issue certs for unverified domains.

**Options.**
- (A) **cert-manager (Kubernetes) Certificate per domain.** Great for *our* fixed domains; awkward for
  thousands of dynamic customer domains (one CRD per domain, reconcile storms, rate limits).
- (B) **Caddy on-demand TLS at the edge/ingress with an `ask` endpoint.** Issues/renews certs lazily on the
  first TLS handshake for a hostname, only if the `ask` endpoint authorizes it.

**Decision.** **Hybrid: Caddy on-demand TLS for customer custom domains; cert-manager for Linkspire's own
fixed domains.**

**Rationale.** On-demand TLS is purpose-built for the "unbounded set of external hostnames" problem: no
per-domain config, certs minted on first use. The **`ask` endpoint queries the Management API and authorizes
only `verified`+`active` domains** (domain onboarding flow, §5 of flows) — this is the control that stops
cert-abuse and keeps us inside ACME rate limits. Cert storage is shared/clustered (e.g. Redis/S3 backend)
so any edge node can serve any issued cert. Our own apex/marketing/API domains are static and best managed
declaratively by cert-manager.

**Consequences.** First request to a brand-new custom domain pays a one-time issuance latency; subsequent
requests are cached. We must monitor ACME issuance rate and implement backoff + an allow/deny cache in the
`ask` path. SNI routing maps hostname → tenant/domain. Apex domains rely on ALIAS/ANAME (A-record fallback
documented as a limitation).

## Assumptions and Open Questions

- **Assumptions:** Vault available for Feistel key + ACME account; edge supports on-demand TLS with a
  shared cert store; CockroachDB is acceptable operationally; dashboard-only TypeScript is acceptable to the
  team.
- **Open questions:** (1) Feistel key-rotation cadence and whether old codes must remain valid forever
  (assumed yes → versioned keys). (2) Do any enterprise tenants require BYO-cert upload instead of ACME?
  (3) Is a managed ClickHouse (ClickHouse Cloud) acceptable in residency regions, or strictly self-hosted?
