<!-- docs/runbooks/README.md -->
# Runbooks

Operational procedures for the incidents the architecture anticipates. Each runbook is structured:
**detect, assess, mitigate, verify, recover, communicate, follow up.** They assume the Phase 6
deployment (Helm on EKS, managed CockroachDB and ClickHouse, ElastiCache, MSK).

| Runbook | When |
|---------|------|
| [`cache-failure.md`](cache-failure.md) | Redis (L2) degraded or down |
| [`db-failover.md`](db-failover.md) | CockroachDB node or region loss |
| [`cert-outage.md`](cert-outage.md) | TLS certificate issuance or renewal failing |
| [`mass-phishing.md`](mass-phishing.md) | Surge of malicious links / abuse |
| [`dsar.md`](dsar.md) | Fulfilling a data subject request |

## Severity and first moves

- **SEV1**: redirects failing broadly, or data at risk. Page on-call immediately, open an incident
  channel, assign an incident commander.
- **SEV2**: degraded but serving (for example cache down, redirects slower). Respond in hours.
- **SEV3**: contained or single-tenant. Handle in business hours.

First moves for any incident: check the dashboards (redirect p99 and error rate, cache hit ratio,
DB health, scanner queue), confirm blast radius, and apply the matching runbook. The binding
constraint never changes: **no synchronous third-party call on the redirect hot path**, and
**never log or persist raw IPs, user agents, or passwords**, not even during an incident.
