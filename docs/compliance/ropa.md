<!-- docs/compliance/ropa.md -->
# Record of Processing Activities (GDPR Art. 30)

> Template, not legal advice. Maintain as a living record; update when processing changes.
> Controller/representative details and DPO: [Legal Entity], [address], [dpo@linkspire.example].

This record covers activities where Linkspire is a **controller** (account operation) and notes,
for completeness, the activities where Linkspire is a **processor** on behalf of customers.

## Activity 1: Account and workspace management (controller)

- **Purpose:** authenticate users, manage workspaces, roles, API keys, and billing identity.
- **Data subjects:** account holders and workspace members.
- **Personal data:** email; Argon2id password hash; encrypted TOTP secret; workspace membership and
  role; API key name, prefix, and hash; chosen residency region.
- **Recipients:** cloud hosting and managed database subprocessors (`subprocessors.md`).
- **Transfers:** none beyond the chosen residency region; SCCs if a restricted transfer occurs.
- **Retention:** life of the account plus a deletion window; see `retention-schedule.md`.
- **Security:** Art. 32 measures in `dpa.md` Annex II.

## Activity 2: Link creation, redirection, and safety scanning (processor for customers)

- **Purpose:** shorten URLs, serve redirects, scan destinations for malware and phishing.
- **Data subjects:** customers' users; indirectly, subjects referenced in destination URLs chosen
  by customers.
- **Personal data:** destination URLs; link settings and scraped public metadata; scan verdicts.
- **Recipients:** managed database and object storage subprocessors; external threat-intelligence
  feeds (hash-prefix lookups) contacted without sending personal data on the hot path.
- **Retention:** links for the life of the account or until deleted; scan results 90 days.
- **Security:** SSRF-guarded scraping; DB CHECK constraints reject dangerous schemes.

## Activity 3: Privacy-safe click analytics (processor for customers)

- **Purpose:** provide customers aggregate link performance without tracking individuals.
- **Data subjects:** end users who click short links.
- **Personal data:** none stored directly. Derived, non-identifying fields only (country, device
  class, browser, OS, referrer domain, source) plus a **daily-rotating** `visitor_hash`. The IP is
  used transiently in memory to derive geography and the hash, then discarded; no raw IP or full
  user agent is persisted.
- **Recipients:** analytics store subprocessor (ClickHouse Cloud).
- **Retention:** raw events 30 days; daily aggregates about 25 months. Cross-day unique identity is
  not computable by design.
- **Security:** salt destroyed within 24 to 48 hours; aggregates exposed to customers, not row-level
  events.

## Activity 4: Security, abuse handling, and audit (controller)

- **Purpose:** detect and act on abuse, maintain a tamper-evident operational trail.
- **Data subjects:** abuse reporters; actors in the system.
- **Personal data:** abuse report contents (short URL and reason); PII-free audit entries (actor
  and API key IDs, action, resource, timestamp).
- **Retention:** audit log 90 days.
- **Security:** append-only log; no IP or UA recorded.

## Activity 5: Data subject request handling (controller and processor)

- **Purpose:** receive and fulfill access, rectification, erasure, restriction, objection, and
  do-not-sell requests.
- **Personal data:** requester identity, request type and regime, status, and statutory due date.
- **Retention:** request records retained as needed to evidence compliance, then expired.
- **Security:** workspace and subject scoping; see `../runbooks/dsar.md`.
