<!-- docs/runbooks/db-failover.md -->
# Runbook: database (CockroachDB) failover

**Severity:** SEV1 if writes are failing or a region is lost. The link store is the RPO=0 source of
truth.

## Detect

- Alerts: CockroachDB node or region unavailable, elevated SQL errors, `/readyz` failing on API and
  workers (they hard-depend on the DB; the redirect service does not for cache hits).

## Assess

- Node loss within a region (CockroachDB self-heals using surviving replicas) versus a full region
  outage.
- Reads vs writes: redirects for cached links keep working regardless; new link creation and the
  dashboard need the DB.

## Mitigate

1. **Single or minority node loss:** no action usually needed. CockroachDB maintains quorum from
   remaining replicas with no data loss (RPO=0). Confirm ranges are up-replicating in the DB
   console.
2. **Region outage:** confirm the cluster retains quorum in the surviving regions. Redirects
   continue from caches; cache-miss lookups for cold links in the lost region are served by other
   replicas. Verify the API and workers reconnect to a healthy gateway.
3. If API/worker pods are pinned to an unhealthy endpoint, roll them:
   `kubectl -n linkspire rollout restart deploy/linkspire-api deploy/linkspire-workers`.
4. Do not fail over to any store that would break global uniqueness of `(domain_id, code)` or
   hostnames. Uniqueness is the database's job and underpins the RPO=0 guarantee; never mint codes
   from a divergent source during an incident.

## Verify

- Writes succeed (create a test link in a scratch workspace), `/readyz` green on API and workers,
  replication healthy, no under-replicated ranges.

## Recover

- When the region returns, let CockroachDB re-replicate before declaring all-clear. Re-enable any
  paused workers.

## Communicate and follow up

- SEV1 comms cadence. Postmortem: quorum configuration, replica placement across regions, and
  whether residency guarantees held throughout.
