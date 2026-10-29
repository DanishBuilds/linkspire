<!-- docs/architecture/05-data-layer.md -->
# Phase 2 — Data Layer

Concrete DDL lives in [`db/migrations/`](../../db/migrations) (CockroachDB, golang-migrate) and
[`db/clickhouse/schema.sql`](../../db/clickhouse/schema.sql). This document explains the distribution
keys, index rationale, migration rules, the Redis key schema, and the multi-region strategy.

## 1. Entity overview

| Table | Purpose | PK / distribution |
|-------|---------|-------------------|
| `workspaces` | tenant root; `home_region` pins residency | `(id)` |
| `users` | accounts; Argon2id hash, MFA, GPC flag | `(id)`, unique `lower(email)` |
| `memberships` | user↔workspace + RBAC role | `(workspace_id, user_id)` |
| `api_keys` | scoped, hashed, rotatable keys | `(id)`, unique `key_hash` |
| `webauthn_credentials` | passkey MFA | `(id)` |
| `domains` | custom domains + verification + TLS authz | `(id)`, unique `hostname` |
| `links` | **hot entity**: code→destination + state | **`(domain_id, code)`** |
| `link_passwords` | isolated Argon2id hashes | `(domain_id, code)` |
| `qr_codes` | QR variants + S3 keys | `(id)`, unique `(domain_id, code, format, style_hash)` |
| `link_metadata` | scraped OG/Twitter/JSON-LD | `(domain_id, code)` |
| `scan_results` | per-scanner verdict history (TTL 90d) | `(id)` |
| `audit_log` | PII-free append-only trail (TTL 90d) | `(id)` |
| `data_subject_requests` | DSAR tracking + SLA timers | `(id)` |
| `retention_policies` | retention config (seeded) | `(data_class)` |
| `reserved_slugs` | vanity blocklist + profanity (seeded) | `(slug)` |
| `idempotency_keys` | safe POST retries (TTL 24h) | `(workspace_id, key)` |
| `webhook_endpoints` | outbound webhooks | `(id)` |

## 2. Distribution / sharding key — the central decision

**Requirement:** "shard on hash of `(domain_id, code)` for links; colocate by tenant for account data."

**How each engine realizes it:**

- **Citus:** `SELECT create_distributed_table('links', 'code');` hash-distributes by the code; account
  tables are distributed by `workspace_id` and **colocated** so a tenant's rows share shards.
- **CockroachDB (chosen, ADR-002):** there is no explicit hash-shard key — data distributes by
  **primary-key range**. The link PK is `(domain_id, code)`.

**Why we do _not_ add `USING HASH` on the links PK.** Hash-sharding exists to spread *monotonic* keys
(timestamps, sequences) that would otherwise hotspot the last range. Our `code` is produced by a **keyed
Feistel permutation (ADR-003)** and is therefore **pseudo-random**. Consecutive link creations land on
*different* ranges automatically, so `(domain_id, code)` is already uniformly distributed. Adding a hash
shard column would (a) be redundant and (b) break clean composite foreign keys, because the hash-sharded
index's uniqueness is on `(shard, domain_id, code)` rather than `(domain_id, code)`.

Keeping a plain PK on `(domain_id, code)` gives us:
1. Even range distribution (from code entropy) — the brief's intent.
2. A directly **FK-referenceable** key, so `link_passwords`, `qr_codes`, `link_metadata`, `scan_results`
   all `FOREIGN KEY (domain_id, code) REFERENCES links` with `ON DELETE CASCADE` — essential for clean
   DSAR erasure.
3. Point-lookup locality on the redirect hot path: `WHERE domain_id = $1 AND code = $2` is a single-range
   PK read.

**Account/tenant colocation:** tenant-scoped tables lead their PK with `workspace_id`
(`memberships (workspace_id, user_id)`) or index it, grouping a tenant's rows on shared ranges.

## 3. Index rationale

| Index | Table | Why it exists |
|-------|-------|---------------|
| PK `(domain_id, code)` | `links` | Hot-path point lookup; even distribution via code entropy |
| `ix_links_workspace (workspace_id, created_at DESC) WHERE deleted_at IS NULL` | `links` | Dashboard listing, newest-first; partial keeps soft-deleted rows out |
| `ix_links_expiring (expires_at) WHERE expires_at IS NOT NULL AND status='active'` | `links` | Expiry sweeper scans only expirable links — tiny index |
| `ix_links_rescan (updated_at) WHERE scan_verdict IN ('pending','suspicious')` | `links` | Scanner re-scan queue; partial = cheap |
| `uq_domains_hostname` | `domains` | **Global** uniqueness of a hostname across all regions |
| `ix_domains_due_check (next_check_at) WHERE state IN ('pending','verified')` | `domains` | Verifier work queue (DNS polling with backoff) |
| `ix_domains_active_host (hostname) STORING (workspace_id,state) WHERE state='active'` | `domains` | O(1) TLS `ask`/SNI authorization lookup; covering via STORING |
| `uq_users_email (lower(email))` | `users` | Case-insensitive unique email via STORED computed column |
| `uq_api_keys_hash` | `api_keys` | Constant-time key verification by hash |
| `ix_dsar_sla (sla_due_at) WHERE status NOT IN ('completed','rejected')` | `data_subject_requests` | SLA breach monitor |
| `uq_qr_variant (domain_id,code,format,style_hash)` | `qr_codes` | Dedup + regen-on-style-change |
| `ix_link_metadata_refresh (refresh_after)` | `link_metadata` | Scraper refresh-TTL queue |

**Partial indexes everywhere** (`WHERE ...`) keep worker-queue indexes proportional to the *work pending*,
not the full corpus — critical at 1.8B rows.

## 4. Migrations & zero-downtime rules

Tooling: **golang-migrate** with versioned, reversible files (`.up.sql`/`.down.sql`) run as a
pre-deploy Kubernetes Job/Helm hook.

Rules enforced:
- **Online by default.** CockroachDB schema changes (ADD COLUMN, CREATE INDEX) are non-blocking; no table
  locks. We still follow expand/contract to decouple schema from app rollout.
- **Expand → migrate → contract.** Add nullable columns / new tables first; backfill; dual-write; switch
  reads; drop the old shape in a *later* release — never rename-in-place.
- **Enum changes isolated.** `CREATE TYPE` is in `000001` alone; any future `ALTER TYPE ... ADD VALUE`
  gets its own migration (CockroachDB forbids using a new enum value in the same transaction that adds it).
- **Reversible.** Every migration ships a `.down.sql`.
- **Multi-region split out.** `000006_multiregion_locality` is prod-only (requires cluster `--locality`
  regions) and is gated so a single-node dev/CI cluster can run `000001`–`000005` unchanged.
- **Retention via storage, not app code.** Row-level TTL (`ttl_expire_after`) on `scan_results` (90d),
  `audit_log` (90d), `idempotency_keys` (24h); the scheduled retention job only reconciles/reports.

## 5. Redis key schema (L2 cache, negative cache, Bloom, rate limits, salt)

All TTLs carry ±10% jitter to prevent synchronized expiry / stampede (except where noted). Memory estimate
ties back to [capacity §5](01-capacity-estimation.md#5-cache-sizing-redis-cluster--valkey--redisbloom).

| Key pattern | Type | Value | TTL | Notes |
|-------------|------|-------|-----|-------|
| `l:{domain_id}:{code}` | String (msgpack) | packed `{dest,status,redirect_type,is_pw,verdict,expires_at}` | 3600s ±jitter | L2 positive cache; ~350 B/entry |
| `n:{domain_id}:{code}` | String | `1` | 60s ±jitter | Negative cache for 404s / enumeration |
| `bf:codes` | RedisBloom | existence filter, `BF.RESERVE 0.001 2e9` | none | ~3.3 GB; rejects nonexistent codes pre-DB |
| `salt:visitor:current` | String | 32-byte random | 48h, rotated daily | Redirect svc hashes IP→visitor_hash then **discards IP** |
| `rl:redir:{fp}` | GCRA/token | counter | window | Redirect-path rate limit by anon fingerprint |
| `rl:create:anon:{fp}` | GCRA | counter | window | Anonymous link-creation budget |
| `rl:create:ws:{workspace_id}` | GCRA | counter | window | Authenticated creation budget |
| `rl:api:{api_key_id}` | GCRA | counter | window | API call budget |
| `pw:att:{domain}:{code}:{fp}` | String | attempt count | progressive | Password brute-force counter |
| `pw:lock:{domain}:{code}:{fp}` | String | `1` | lockout | Temporary lockout after threshold |
| `dom:authz:{hostname}` | String | `active`/`deny` | 300s | TLS `ask`-endpoint decision cache |
| `sf:{domain}:{code}` | String (SET NX) | `1` | 2s | Optional cross-pod singleflight lock (L1 singleflight is primary) |

**Memory (restated):** positive ≈ 10.5 GB (30M hot entries) + negative ≈ 0.4 GB + Bloom ≈ 3.3 GB ≈
**14.2 GB** data; provision ~2.2× ⇒ **~32 GB usable** across **3 primary + 3 replica** 16 GB nodes.

**Privacy note:** the visitor salt is the only place an IP is transiently touched, in memory, inside the
redirect service — the salted hash is computed and the IP is dropped before any event leaves the process.
The salt is destroyed within 24–48h so `visitor_hash` cannot correlate a visitor across days.

## 6. ClickHouse analytics (summary)

See [`db/clickhouse/schema.sql`](../../db/clickhouse/schema.sql).

- `clicks_raw` — `MergeTree`, daily partitions, **TTL 30 days**. No IP, no UA columns — only
  country/region/device/browser/os/referrer-domain/source + a daily-salted `visitor_hash`.
- `clicks_daily` — `AggregatingMergeTree`, monthly partitions, **TTL 760 days (~25 months)**; fed by
  `clicks_daily_mv`; exposes `countMerge`/`uniqCombinedMerge` via the `clicks_daily_final` view.
- Ingestion: `clicks_queue` (Kafka engine, consumes Redpanda `click_events`) → `clicks_queue_mv` →
  `clicks_raw`.
- **Privacy > accuracy tradeoff (documented):** because the salt rotates daily, unique-visitor counts are
  per-day only; multi-day/cross-day unique visitors are **not computable by design**. Opt-in alternative:
  a tenant may enable longer-lived (e.g. 7-day) salts to get rolling-week uniques — strictly opt-in and
  surfaced in the privacy settings.

## 7. Multi-region & global consistency

- **Global uniqueness (no duplicate codes/hostnames across regions):** enforced by CockroachDB's globally
  consistent unique indexes — links PK `(domain_id, code)`, `domains.hostname`, `users.email_normalized`.
  A concurrent create in US and EU for the same `(domain_id, code)` cannot both commit; one fails the
  unique constraint. This is the **RPO = 0** guarantee for link identity.
- **Residency / write-home:** `REGIONAL BY ROW` pins each tenant's rows to its `home_region` (`us`/`eu`);
  the app sets `crdb_region` from the workspace on insert. Reads/writes for a tenant are served locally.
- **Failover:** `SURVIVE REGION FAILURE` keeps quorum across a full region loss; CockroachDB auto-promotes
  replicas — **RTO < 5 min** with no data loss. Read replicas per region serve cache misses; the redirect
  path tolerates a replica/region outage per the [degradation matrix](02-architecture-and-flows.md#7-degradation-matrix).
- **Reference data** (`reserved_slugs`, `retention_policies`) is `GLOBAL` for low-latency local reads
  everywhere.

## Assumptions and Open Questions

- **Assumptions (defaults chosen as agreed):** burst is transient (no change to node counts); CCPA
  analytics erasure is satisfied by **anonymization** (dropping join keys / `visitor_hash`) rather than
  physical row deletion, preserving aggregate counts; **active-passive per-tenant write-home** (not
  active-active writes). Two regions to start (`us`, `eu`); region names in `000006` are placeholders to
  edit per environment.
- **Open questions:** (1) Max stored destination length is capped at 2048 — confirm product limit.
  (2) Should `click_count` on `links` be removed entirely in favor of a ClickHouse-sourced check for
  click-limit enforcement, to avoid any hot-row write? (current design batches updates). (3) Do enterprise
  tenants need a third residency region (e.g. APAC) at launch? (4) Confirm 90-day TTL on `audit_log` meets
  the longest abuse-dispute window you must support.
