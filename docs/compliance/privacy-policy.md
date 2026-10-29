<!-- docs/compliance/privacy-policy.md -->
# Privacy Policy (template)

> Template, not legal advice. Review with counsel before publishing. Replace every bracketed
> placeholder. Effective date: [date]. Controller: [Legal Entity], [address], contact
> [privacy@linkspire.example], Data Protection Officer [dpo@linkspire.example].

## 1. Who we are and this policy's scope

Linkspire operates a privacy-first URL shortening and link-analytics platform. This policy explains
what personal data we process when you (a) hold an account, (b) are a member of a workspace, (c)
click a Linkspire short link, or (d) contact us or report abuse.

For business customers, when we shorten links and produce analytics on their behalf, they are the
**controller** and we are the **processor**; our commitments to them are in the Data Processing
Agreement (`dpa.md`). This policy covers the data for which **we** are the controller (primarily
account data and our own service operation).

## 2. What we collect, and what we deliberately do not

### Account and workspace data (you give us)

- Email address, hashed password (Argon2id; we never store or log the plaintext), and optional
  time-based one-time-password (TOTP) secret, stored encrypted.
- Workspace name and chosen data residency region (United States or European Union).
- API key metadata (name, prefix, and a SHA-256 hash of the key; never the key itself after
  creation).

### Link and redirect data

- The destination URLs you shorten, link settings, and optional metadata (title, description,
  image) scraped from public destination pages.

### Click analytics (privacy-engineered)

When someone clicks a short link, we derive **in memory**, then immediately discard the IP address:

- Country (ISO-3166 alpha-2), device class, browser family, OS family, referrer domain (host only,
  path and query stripped), and traffic source (web, QR, or API).
- A `visitor_hash` computed with a **salt that rotates daily and is destroyed within 24 to 48
  hours**, used for same-day unique counts only.

We do **not** store raw IP addresses, full user-agent strings, precise geolocation, or any
cross-day visitor identifier. Analytics are daily aggregates. This is a deliberate accuracy
tradeoff in favor of privacy, and it is enforced in code.

### Abuse reports

If you report a link, we process the short URL and the reason you provide to review and act on it.

## 3. Why we process it (lawful bases, GDPR Art. 6)

| Purpose | Lawful basis |
|---------|--------------|
| Providing accounts, links, and redirects | Performance of a contract (Art. 6(1)(b)) |
| Security, abuse prevention, malware and phishing scanning | Legitimate interests (Art. 6(1)(f)) |
| Privacy-safe aggregate analytics | Legitimate interests, with no cross-day tracking and an opt-out |
| Meeting legal obligations (for example responding to lawful requests) | Legal obligation (Art. 6(1)(c)) |

## 4. How long we keep it

See the full `retention-schedule.md`. In summary: raw click events are deleted after 30 days,
daily aggregates after about 25 months, security and audit logs after 90 days, and account data
for the life of the account plus a short wind-down window after deletion.

## 5. Who we share it with

We do not sell or share personal information. We use the vetted subprocessors listed in
`subprocessors.md` (cloud hosting, managed database, and analytics storage) under contract. We
disclose data to authorities only when legally compelled.

## 6. International transfers

You choose a residency region (US or EU) for your workspace, and we keep workspace data in that
region. Where a transfer between regions is unavoidable, we rely on appropriate safeguards such as
Standard Contractual Clauses.

## 7. Your rights

Depending on where you live (GDPR, UK GDPR, CCPA/CPRA, and similar laws), you may request access,
rectification, erasure, restriction, portability, or objection, and you may opt out of any sale or
sharing. We respond within 30 days (GDPR) or 45 days (CCPA/CPRA). File a request from your account
settings (Privacy), or email [privacy@linkspire.example]. Verified requests are processed by our
automated DSAR workflow; see the operational `../runbooks/dsar.md`.

### California (CCPA/CPRA) notice

We do not sell or share personal information and do not use it for cross-context behavioral
advertising. You can still record a "Do Not Sell or Share" preference in your account; we honor the
Global Privacy Control (GPC) signal. You have the rights to know, delete, correct, and to
non-discrimination for exercising them.

## 8. Security

Passwords and link passwords are hashed with Argon2id. Transport is TLS only. API tokens are
short-lived JWTs that carry no personal data, and API keys are stored only as hashes. Outbound
fetches (metadata scraping, webhooks) run through an SSRF guard. See `docs/architecture/04-threat-model.md`.

## 9. Children

The service is not directed to children under 16, and we do not knowingly collect their data.

## 10. Changes and contact

We will post changes here with a new effective date. Questions: [privacy@linkspire.example];
EU/UK representative [if applicable]; supervisory authority complaints are your right at any time.
