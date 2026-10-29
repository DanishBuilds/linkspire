<!-- docs/compliance/README.md -->
# Compliance pack

Working documents that turn Linkspire's privacy-by-design architecture into the records and
agreements a data protection program expects. They are grounded in what the code actually does
(see `docs/architecture/`), not aspirational copy.

> **Not legal advice.** These are engineering-authored templates. Have qualified privacy counsel
> review and adapt every document before it is published or signed. Bracketed fields such as
> `[Legal Entity]`, `[address]`, and `[dpo@linkspire.example]` are placeholders for your own
> details. "Linkspire" is the project working name.

| Document | Purpose |
|----------|---------|
| [`privacy-policy.md`](privacy-policy.md) | Public notice to data subjects (GDPR Arts. 13 to 14, CCPA/CPRA) |
| [`dpa.md`](dpa.md) | Controller to processor Data Processing Agreement for business customers |
| [`ropa.md`](ropa.md) | Record of Processing Activities (GDPR Art. 30) |
| [`retention-schedule.md`](retention-schedule.md) | Every data class, its store, TTL, and legal basis |
| [`subprocessors.md`](subprocessors.md) | Third parties that process personal data on our behalf |
| [`dpia.md`](dpia.md) | Data Protection Impact Assessment for the link and analytics processing |

## The three claims the whole program rests on

1. **No raw IP, full user agent, or password is ever persisted.** Country, device class, browser,
   and OS are derived in memory inside the redirect service and the IP is discarded before any
   event leaves the process (`docs/architecture/05-data-layer.md`).
2. **Visitors cannot be tracked across days.** `visitor_hash` uses a salt that rotates daily and
   is destroyed within 24 to 48 hours, so cross-day unique counts are not computable by design.
3. **Retention is enforced by storage, not hope.** Row and partition TTLs expire data on schedule;
   the retention worker only reconciles and reports (see `retention-schedule.md`).

These are verifiable in code, which is what makes the records below defensible.
