<!-- docs/compliance/subprocessors.md -->
# Subprocessors

> Template, not legal advice. Keep this list current and notify customers before adding or replacing
> a subprocessor (see `dpa.md` Section 4). Confirm each entry's current DPA, region options, and
> certifications before relying on them.

Linkspire engages the following subprocessors to process personal data on behalf of customers. Each
is bound by data protection terms no less protective than our DPA.

| Subprocessor | Service to Linkspire | Personal data processed | Region(s) |
|--------------|----------------------|-------------------------|-----------|
| Amazon Web Services | Compute (EKS), cache (ElastiCache Redis), streaming (MSK), image registry (ECR), key management (KMS), logging (CloudWatch) | All platform data at rest/in transit within the chosen region | Customer's residency region (US or EU) |
| CockroachDB Cloud | Managed distributed SQL (accounts, links, operational data) | Account email, link destinations, operational records | US or EU, matching residency |
| ClickHouse Cloud | Managed analytics store | Derived, non-identifying click aggregates and daily-salted hashes only | US or EU, matching residency |
| External threat-intelligence feeds (for example URLHaus, Safe Browsing hash-prefix) | Malware and phishing signals | None on the hot path; hash-prefix lookups do not transmit personal data | Global |

Notes:

- The redirect hot path makes **no synchronous third-party call**, so no end-user data is sent to
  any subprocessor at click time; analytics events are written to our own managed stores.
- Threat-intelligence lookups use local hash-prefix databases and SSRF-guarded fetches; they are
  listed for completeness and do not receive personal data.
- Container images are distributed via the registry only; images contain no personal data.

Customer notification channel for subprocessor changes: [status.linkspire.example or email list].
