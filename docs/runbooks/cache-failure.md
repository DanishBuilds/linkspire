<!-- docs/runbooks/cache-failure.md -->
# Runbook: cache (Redis) failure

**Severity:** usually SEV2. Redirects keep serving; they get slower and push load to the database.

## Detect

- Alerts: Redis unreachable, cache hit ratio collapse, redirect p99 rising toward the DB latency.
- The redirect service logs L2 errors but continues (Redis is a soft dependency, excluded from
  readiness on purpose).

## Assess

- Is it total (ElastiCache primary down) or partial (one replica, a network blip)?
- Check DB load. The real risk is a stampede of cache misses hitting CockroachDB.

## Mitigate

1. The service already degrades gracefully: in-process L1 LRU still serves hot links, singleflight
   collapses duplicate misses, and the Bloom filter short-circuits lookups for codes that do not
   exist. Confirm these are doing their job (hit ratio on L1, singleflight metric, DB QPS).
2. If ElastiCache failed over, confirm the new primary and that the service reconnected. If not,
   roll the redirect pods to force reconnection:
   `kubectl -n linkspire rollout restart deploy/linkspire-redirect`.
3. If the DB is approaching its limits, scale the redirect tier up (HPA should react; raise
   `maxReplicas` temporarily if needed) and consider temporarily increasing L1 size
   (`L1_SIZE`) to hold more hot links in process.
4. Do not disable the SSRF guard, scanning, or any privacy control to "save CPU".

## Verify

- Redirect p99 back under target, error rate near zero, DB load stable, cache hit ratio recovering
  as Redis returns.

## Recover

- Once Redis is healthy, hit ratio climbs on its own as keys repopulate (TTLs carry jitter to avoid
  a synchronized refill). Return any temporary HPA or L1 overrides to defaults.

## Communicate and follow up

- Status page if customer-visible latency occurred. Postmortem: was L1 sized adequately, did the
  Bloom filter hold, did autoscaling react in time.
