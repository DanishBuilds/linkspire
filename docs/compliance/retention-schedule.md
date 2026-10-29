<!-- docs/compliance/retention-schedule.md -->
# Data retention schedule

> Template, not legal advice. Retention is enforced primarily by storage-layer TTLs and partition
> drops; the retention worker reconciles and reports coverage, it does not decide policy.
> Authoritative config lives in the `retention_policies` table (keyed by `data_class`).

## Click analytics (ClickHouse)

| Data class | Store | Retention | Mechanism | Notes |
|------------|-------|-----------|-----------|-------|
| Raw click events | `clicks_raw` | 30 days | `TTL event_time + INTERVAL 30 DAY`, daily partitions | No IP or UA columns exist; only derived fields + daily `visitor_hash` |
| Daily aggregates | `clicks_daily` | ~760 days (~25 months) | `TTL event_date + INTERVAL 760 DAY` | Aggregate states only; no per-visitor rows |

## Operational data (CockroachDB)

| Data class | Table | Retention | Mechanism | Notes |
|------------|-------|-----------|-----------|-------|
| Security scan verdicts | `scan_results` | 90 days | row TTL (`ttl_expire_after`) | Per-scanner history |
| Audit trail | `audit_log` | 90 days | row TTL | PII-free, append-only |
| Idempotency keys | `idempotency_keys` | 24 hours | row TTL | Safe POST retries only |
| Link metadata | `link_metadata` | Refreshed on a TTL; removed with the link | scraper refresh queue | Public page metadata |
| DSAR records | `data_subject_requests` | Kept to evidence compliance, then expired | retention worker | Statutory-proof window |

## Account and content data (CockroachDB)

| Data class | Table(s) | Retention |
|------------|----------|-----------|
| Accounts and auth | `users`, `webauthn_credentials` | Life of account; purged after the deletion window on account closure or erasure |
| Workspaces and membership | `workspaces`, `memberships` | Life of account |
| Links and settings | `links`, `link_passwords`, `qr_codes`, `domains` | Until deleted by the customer, or account-deletion window |
| API keys | `api_keys` | Until revoked; metadata retained for audit, key hash removed on deletion |
| Webhooks | `webhook_endpoints`, `webhook_deliveries` | Endpoints until removed; deliveries short-lived |

## Transient state (Redis)

| Data class | Key pattern | Retention | Notes |
|------------|-------------|-----------|-------|
| Visitor salt | `salt:visitor:current` | 48 hours, rotated daily | The only place an IP is transiently touched, in memory; IP discarded immediately |
| Hot link cache / negative cache | cache keys | Minutes to hours with jitter | Derived from committed data |
| Rate-limit counters | rate-limit keys | Short windows | No personal identifiers stored |

## Account deletion and erasure

On account closure or a verified erasure request, account, workspace, link, and key data are
deleted within [30] days (`dpa.md` Section 9). Analytics require no action because they are already
non-identifying and expire on the TTLs above. The retention worker verifies TTL coverage and reports
any data class lacking a policy.

## Open item carried from Phase 1

Confirm that the 90-day `audit_log` TTL satisfies the retention obligations of the target markets;
adjust the `retention_policies` row if a longer statutory minimum applies.
