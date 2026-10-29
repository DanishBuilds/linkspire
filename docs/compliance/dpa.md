<!-- docs/compliance/dpa.md -->
# Data Processing Agreement (template)

> Template, not legal advice. This DPA is entered into between the customer ("Controller") and
> [Legal Entity] ("Processor", "Linkspire") and supplements the main service agreement. Review and
> adapt with counsel. Bracketed fields are placeholders.

## 1. Roles and scope

The Controller determines the purposes and means of processing the personal data it submits to the
Service. Linkspire processes that data only as a Processor, solely to provide the Service and on the
Controller's documented instructions (this DPA and the service agreement being such instructions).

## 2. Subject matter, duration, nature, and purpose (GDPR Art. 28(3))

- **Subject matter:** URL shortening, redirection, and privacy-safe link analytics.
- **Duration:** the term of the service agreement plus the deletion window in Section 9.
- **Nature and purpose:** storing destination URLs and link settings; serving redirects; deriving
  and aggregating non-identifying click metrics; malware and phishing scanning; QR generation.
- **Categories of data subjects:** the Controller's authorized users and end users who click the
  Controller's links.
- **Categories of personal data:** account identifiers (email) of the Controller's users; link
  destinations that may contain identifiers chosen by the Controller; and, for click events,
  **derived, non-identifying** fields only (country, device class, browser, OS, referrer domain,
  source, and a daily-rotating `visitor_hash`). No raw IP addresses, full user agents, or special
  category data are processed by design. The Controller agrees not to place special category data
  in link destinations.

## 3. Processor obligations (Art. 28(3))

Linkspire shall: (a) process only on documented instructions, including for transfers; (b) ensure
persons authorized to process are under confidentiality obligations; (c) implement the measures in
Annex II; (d) respect the conditions for engaging subprocessors (Section 4); (e) assist the
Controller with data subject requests (Section 5); (f) assist with security, breach notification,
and DPIAs (Sections 6 to 7); (g) delete or return data on termination (Section 9); and (h) make
available information necessary to demonstrate compliance and allow audits (Section 10).

## 4. Subprocessors

The Controller provides general authorization for the subprocessors listed in `subprocessors.md`.
Linkspire imposes data protection terms on each subprocessor no less protective than this DPA and
remains liable for their performance. Linkspire will give at least [30] days' notice of any
intended addition or replacement, during which the Controller may object on reasonable data
protection grounds.

## 5. Data subject rights

Taking into account the nature of the processing, Linkspire assists the Controller with appropriate
technical and organizational measures to respond to data subject requests. Account holders can
exercise rights directly through the dashboard; the automated DSAR workflow handles access,
rectification, erasure, restriction, objection, and do-not-sell within the statutory windows
(30 days GDPR, 45 days CCPA/CPRA). See `../runbooks/dsar.md`.

## 6. Security (Art. 32)

Linkspire implements and maintains the technical and organizational measures in Annex II, including
Argon2id password hashing, encryption in transit, encryption at rest for managed stores, hashed API
keys, SSRF-guarded outbound requests, least-privilege and non-root workloads, and the data
minimization described in Annex I.

## 7. Personal data breach (Art. 33)

Linkspire notifies the Controller without undue delay and in any case within [72] hours of becoming
aware of a personal data breach affecting the Controller's data, with the information the Controller
reasonably needs to meet its own obligations. See `../runbooks/` for the operational response.

## 8. International transfers

Workspace data is stored in the residency region the Controller selects (US or EU). Where a
restricted transfer occurs, the parties rely on the EU Standard Contractual Clauses (and the UK
Addendum where applicable), which are incorporated by reference, with Linkspire acting as data
importer.

## 9. Return and deletion

On termination, Linkspire deletes the Controller's personal data within [30] days, unless retention
is required by law, after which TTL-based expiry removes any residual copies on schedule (see
`retention-schedule.md`). On request within that window, Linkspire returns the data in a commonly
used format.

## 10. Audits

Linkspire makes available compliance documentation (this pack, architecture records, and available
third-party attestations of its subprocessors) and supports audits at reasonable intervals, subject
to confidentiality and minimizing disruption.

---

## Annex I: Details of processing

Covered in Sections 2. Data minimization summary: destination URLs and account email are the only
directly identifying fields; all click analytics are derived in memory with the IP discarded before
any event is emitted, and no cross-day visitor identifier exists.

## Annex II: Technical and organizational measures (Art. 32)

| Measure | Implementation |
|---------|----------------|
| Pseudonymization and minimization | No raw IP/UA stored; daily-rotating, self-destructing visitor salt; aggregates only |
| Encryption | TLS in transit; at-rest encryption on managed DB, cache, and streaming; envelope-encrypted TOTP secrets |
| Credential protection | Argon2id (m=65536, t=3, p=2) for passwords and link passwords; API keys stored as SHA-256 hashes |
| Access control | Workspace-scoped RBAC (owner, admin, member, viewer); short-lived JWTs with no PII; optional TOTP MFA |
| Network protection | SSRF guard on all outbound fetches; non-root, read-only-root-filesystem containers; least-privilege IAM |
| Resilience | Multi-AZ managed data stores; RPO=0 link store; documented failover runbooks |
| Auditability | PII-free append-only audit log (90-day retention) |
| Secure development | SAST, dependency and image scanning, SBOMs in CI (`.github/workflows/`) |

## Annex III: Subprocessors

See `subprocessors.md`.
