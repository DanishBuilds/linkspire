<!-- docs/architecture/01-capacity-estimation.md -->
# Phase 1.1 — Capacity Estimation

All sizing decisions below are derived from the non-functional targets, with the arithmetic shown so
each number is auditable. Where a factor is a judgement call (e.g. cache hit ratio, compression), it is
stated explicitly and chosen conservatively.

## 0. Inputs (restated)

| Parameter | Value |
|-----------|-------|
| Redirects/day (avg) | 100,000,000 |
| Burst peak | 50,000 req/s |
| Read:write ratio | ~100:1 |
| New links/day | 1,000,000 |
| Redirect latency | p50 < 10 ms, p99 < 30 ms at origin; p99 < 5 ms on in-service cache hit |
| Availability | 99.99% redirect, 99.9% dashboard/API |
| Durability | RPO = 0 for link data; RTO < 5 min/region |
| Retention horizon for storage math | 5 years |

## 1. Request rate

```
avg redirect QPS   = 100,000,000 / 86,400 s        = 1,157 req/s
diurnal peak (~4x) ≈ 4,000 – 4,600 req/s  (consumer traffic is spiky; 3–4x mean is typical)
design burst       = 50,000 req/s         (given; ~11x avg → viral link / campaign spike headroom)

writes (avg)       = 1,157 / 100          ≈ 11.6 writes/s   (matches 1M links/day = 11.57/s)
writes (burst)     ≈ 50,000 / 100         ≈ 500 writes/s
bulk-create spikes ≈ 2,000–5,000 writes/s (short, API-driven; absorbed by queue + idempotency)
```

**Implication:** the system is overwhelmingly read-dominated. The write tier is tiny and can run on a
single logical primary per region; **all engineering effort and cost goes into the read (redirect) path.**

## 2. Short-code key space (collision math)

```
Base62, length 7  → 62^7 = 3,521,614,606,208 ≈ 3.52 × 10^12 codes per domain

5-year link count  = 1,000,000/day × 365 × 5 = 1.825 × 10^9 links
fill ratio         = 1.825e9 / 3.52e12       = 0.052 %   (per domain; most load is on 1–2 system domains)
```

At 1M links/day a single domain exhausts < 0.06% of the space in 5 years; the space lasts **> 9,000
years** at this rate. **Length 7 is sufficient**; we do not need length 8. Codes are produced by a keyed
Feistel permutation over the `62^7` space (see [ADR-003](03-adr.md#adr-003--short-code-id-generation)),
so they are collision-free by construction and non-enumerable — no probabilistic birthday collisions.

## 3. Primary (link) storage — 5 years

Row-size estimate for `links` (dominant table). Destination URL dominates; we cap stored length at
2 KB but the *average* URL is ~120 bytes.

```
id            bigint        8
workspace_id  bigint        8
domain_id     bigint        8
code          varchar(7)   ~16 (data + header)
destination   text        ~120 (avg; p95 ~400)
status        smallint      2
redirect_type smallint      2
scan_verdict  smallint      2
created_at    timestamptz   8
updated_at    timestamptz   8
expires_at    timestamptz   8
click_limit   bigint        8
creator_id    bigint        8
flags/bitset  int           4
tuple header + null bitmap + alignment  ~40
                              ----
data/row                   ≈ 250 bytes
```

```
index overhead (PK + unique(domain_id,code) + workspace_id + partial expires_at)  ≈ 1.5× data
bloat/fill-factor + MVCC/replication amplification cushion                        ≈ 1.0× data
effective per link                 ≈ 250 × 2.5                 ≈ 625 bytes

logical link storage (5 yr)        = 1.825e9 × 625 B           ≈ 1.14 TB
replicated (RF = 3, distributed SQL) ≈ 3.4 TB raw
+ 30% operational headroom          ≈ 4.5 TB raw cluster-wide
```

Other Postgres tables (users, workspaces, memberships, domains, api_keys, audit_log, data_subject_requests,
retention_policies, link_passwords, qr_codes metadata) are **orders of magnitude smaller** than `links`
and are rounded into the 30% headroom. `scan_results` grows with re-scans but is capped by retention.

## 4. Click-event pipeline (Kafka/Redpanda + ClickHouse)

### 4.1 Stream (Redpanda)

```
events/day        = 100,000,000  (1 click event per redirect)
avg event QPS     = 1,157/s ; burst = 50,000/s
serialized size   ≈ 150–200 bytes (protobuf; no raw IP/UA — see privacy)

ingress (avg)     = 1,157 × 200 B      ≈ 231 KB/s
ingress (burst)   = 50,000 × 200 B     ≈ 10 MB/s        → trivial for a 3-broker cluster
daily volume      = 100e6 × 200 B      ≈ 20 GB/day
buffer 72 h, RF=3 = 20 × 3 × 3         ≈ 180 GB on disk  → small
```

A **3-broker Redpanda** cluster per region is sized by durability/HA, not throughput. Partitions chosen
for consumer parallelism (e.g. 24) not load.

### 4.2 Analytics store (ClickHouse)

```
raw event logical row   ≈ 150 bytes
columnar compression    ≈ 6× (conservative; low-cardinality dims compress 10–50×)
compressed/event        ≈ 25 bytes
raw compressed/day      = 100e6 × 25 B        ≈ 2.5 GB/day
raw retention 30 days   = 2.5 × 30            ≈ 75 GB
aggregates (25 months)  : daily rollups per (link, country, device_class, browser, os, referrer_domain)
                          cardinality bounded; ≈ hundreds of GB worst case over 25 months
ClickHouse cluster (RF=2) : ~0.5–1 TB usable  → 2–3 nodes, NVMe
```

ClickHouse enforces retention via **native TTLs** (raw 30 d → drop; rollups 25 months → drop); see Phase 2.

## 5. Cache sizing (Redis Cluster / Valkey + RedisBloom)

The cache holds the **hot working set**, not the full corpus. Access follows a power law.

### 5.1 Positive cache (code → link record)

```
assume daily unique codes accessed ≈ 15M  (of 1.825e9 total; heavy skew)
cache window sized slightly larger  ≈ 30M hot entries

per entry:
  key  "l:{domainId}:{code}"           ≈ 24 B
  value packed{dest,status,type,flags,exp} ≈ 200 B
  Redis per-key overhead (dictEntry, robj, expire bucket) ≈ 90–120 B
  ----------------------------------------
  ≈ 350 B/entry

positive cache = 30M × 350 B           ≈ 10.5 GB
```

### 5.2 Negative cache (nonexistent codes)

```
bounded, short TTL (e.g. 60 s), jittered; absorbs enumeration bursts
≈ 5M entries × 80 B                     ≈ 0.4 GB
```

### 5.3 Existence Bloom filter (RedisBloom) — anti-enumeration

Bits/item for false-positive rate p:  `m/n = -ln(p) / (ln 2)^2`

```
p = 1%    → 9.59 bits/item → 1.825e9 × 9.59 / 8 ≈ 2.19 GB
p = 0.1%  → 14.38 bits/item → 1.825e9 × 14.38 / 8 ≈ 3.28 GB   ← chosen
```

A **0.1% FPR Bloom** over all existing codes costs **~3.3 GB** and lets the hot path reject the vast
majority of enumeration/scan lookups before touching Redis GET or Postgres. (A Bloom false positive only
causes one extra cache/DB lookup — never a wrong redirect.)

### 5.4 Redis totals and node count

```
data = 10.5 (positive) + 0.4 (negative) + 3.3 (bloom)   ≈ 14.2 GB
provision at ≤ 65% maxmemory util + replication buffers + rehash/fragmentation ≈ 2.2×
usable to provision                                      ≈ 32 GB

topology: 3 primary shards (~11 GB data each pre-overhead) + 1 replica each = 6 nodes
node class: 16 GB RAM  (keeps steady-state util < 70%)
```

**Throughput check:** Redis-facing QPS = `(1 − L1_hit) × redirect_QPS`. With L1 hit ratio **70–85%** on
the heavy-hitter links, burst Redis load ≈ `(0.15–0.30) × 50,000 = 7,500–15,000 ops/s`, spread over 3
shards = **2.5–5k ops/s/shard**. A single Redis node sustains **>100k ops/s**. **Redis is memory-bound,
not CPU-bound** — node count is driven by §5.4 memory, with huge throughput headroom.

### 5.5 L1 in-process LRU (per redirect pod)

```
100,000 entries × ~300 B                ≈ 30 MB per pod   (negligible vs 256–512 MB pod)
```

Because the top links drive most QPS and exhibit strong temporal locality, an L1 of 100k entries captures
the bulk of traffic. L1 TTLs are short + jittered to bound staleness and prevent synchronized expiry.

## 6. Compute (pod/node counts)

### 6.1 Redirect service (Go)

```
per-pod sustained budget (conservative, to hold p99): 5,000 rps  (a tuned Go pod can do 10–20k on cache hits)
burst requirement        = 50,000 rps
pods for burst           = 50,000 / 5,000 = 10
run min 12 (N+2, across 3 AZs); HPA max 30 (deploys, AZ loss, cold cache)
per pod                  ≈ 0.5 vCPU request / 1 vCPU limit, 256–512 MB
```

### 6.2 Management API + workers (Go)

```
API write/read QPS small (<1k steady). min 3 pods (HA across AZs), HPA to ~10.
workers (scanner/scraper/qr/verifier/retention/dsar): queue-driven, scale on lag.
  scraper runs in an isolated egress-restricted node pool (SSRF containment).
```

### 6.3 Distributed SQL (CockroachDB — see ADR-002)

```
write QPS: ≤500/s burst → trivial.
read QPS to DB = cache MISSES only. At steady 99%+ aggregate hit, DB reads ≈ 500/s at burst;
  provision for 5–10k rps read capacity for cold-cache / incident scenarios.
storage-bound: 4.5 TB raw (§3) → min 3 nodes/region (RF=3), scale to 6–9 as data grows.
multi-region: 3 regions × 3 nodes = 9 nodes baseline, NVMe, each ~0.75–1.5 TB working set.
```

### 6.4 PgBouncer / pooling

CockroachDB has a native connection broker, but we still front coordinators with **2 PgBouncer per region
(HA, transaction mode)** so thousands of client connections fan into a few hundred server connections.

### 6.5 Bandwidth / egress

```
redirect response (3xx + Location) < 1 KB
burst egress = 50,000 × 1 KB = 50 MB/s ≈ 400 Mbps   → comfortable
CDN absorbs cacheable statuses (301/308) and short-TTL edge caching for 302; QR assets served from S3+CDN.
```

## 7. Availability budgets

```
99.99% → 52.6 min/year downtime budget (redirect path)
99.9%  → 8.77 h/year (dashboard/API)
```

Met via: multi-AZ + multi-region active-active redirect, anycast/edge DNS health-checked failover, every
stateless tier N+2 with PodDisruptionBudgets, and the degradation modes in
[`02-architecture-and-flows.md`](02-architecture-and-flows.md#7-degradation-matrix) that keep redirects
serving even when Redis/Kafka/scanners are down. RTO < 5 min/region is met by pre-provisioned standby
capacity and automated DB region failover.

## 8. Summary sizing table

| Resource | Driver | Sizing |
|----------|--------|--------|
| Redirect pods | 50k burst @ 5k/pod + N+2 | min 12, HPA → 30 |
| API pods | HA | min 3, HPA → 10 |
| Redis Cluster | 14.2 GB hot set + bloom | 6 nodes (3 primary/3 replica), 16 GB each |
| RedisBloom | 1.825e9 codes @ 0.1% FPR | ~3.3 GB |
| CockroachDB | 4.5 TB raw, 3 regions, RF=3 | 9 nodes baseline (3/region) |
| Redpanda | HA + 72h buffer | 3 brokers/region |
| ClickHouse | 75 GB raw + rollups, RF=2 | 2–3 nodes/region |
| Object storage (QR) | 1M links × (SVG+PNG) ~ few KB | S3 + CDN, lifecycle rules |

## Assumptions and Open Questions

- **Assumptions:** L1 hit ratio 70–85%; aggregate cache hit ≥ 99% steady-state; average destination URL
  ~120 B; ClickHouse compression ~6×; Redpanda event ~200 B; 15M daily-unique hot codes; 3 regions.
- **Open questions:** (1) Is the 50k burst *sustained* for minutes or transient seconds? Sustained changes
  HPA max and Redis headroom. (2) Expected ratio of custom-domain traffic vs system-domain — affects edge
  TLS cert volume and cache-key cardinality. (3) Hard cap on stored destination length (assumed 2 KB).
  (4) Required number and geography of residency regions beyond EU/US.
