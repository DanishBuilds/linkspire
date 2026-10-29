<!-- docs/compliance/dpia.md -->
# Data Protection Impact Assessment (template)

> Template, not legal advice. A DPIA is advisable because the service involves large-scale
> processing of behavioral (click) data. This assessment reflects the privacy-by-design choices in
> `docs/architecture/`. Review with counsel; record sign-off below.

## 1. Description of the processing

Linkspire shortens URLs, serves high-volume redirects, and produces aggregate link analytics.
Scale target: about 100 million redirects per day. The analytics processing is the focus of this
assessment because it concerns the behavior of end users who click links.

Data flow: a click reaches the redirect service, which derives country, device class, browser, OS,
referrer domain, and source **in memory**, computes a `visitor_hash` from the IP and a daily salt,
then **discards the IP** before emitting an event. Events are aggregated daily and expire on TTLs.

## 2. Necessity and proportionality

Link owners have a legitimate need to understand link performance. The processing is proportionate
because it is minimized to non-identifying, aggregate metrics and deliberately avoids the identifying
data (raw IP, full user agent, precise location, cross-day identifiers) that conventional analytics
collect. Accuracy is traded for privacy where they conflict, and an opt-out is offered.

## 3. Risks and mitigations

| Risk | Likelihood / impact | Mitigation | Residual |
|------|---------------------|------------|----------|
| Re-identification of visitors | Low / high | No IP or UA stored; `visitor_hash` salt rotates daily and self-destructs in 24 to 48h, so cross-day linkage is not possible; only aggregates are exposed | Low |
| Function creep (richer tracking added later) | Medium / high | Minimization enforced in schema and code (no IP/UA columns exist); change would require schema change and this DPIA's revision | Low |
| Breach of stored data | Low / medium | Encryption at rest and in transit; the most sensitive stored analytics field is a non-reversible daily hash | Low |
| SSRF via destination scraping | Low / high | SSRF guard: DNS pinning, blocked metadata/private/encoded addresses, redirect and size caps; see `04-threat-model.md` | Low |
| Abuse (phishing/malware links) harming third parties | Medium / medium | Synchronous fast-path scan at creation plus asynchronous re-scan; abuse report intake and review | Medium |
| Excessive retention | Low / medium | Storage-enforced TTLs (30 days raw, ~25 months aggregate) and a reconciliation worker | Low |

## 4. Consultation and conclusion

Stakeholders: engineering, security, and privacy counsel [names]. The residual risks are acceptable
given the minimization built into the design. Reassess this DPIA when the analytics schema, the
retention periods, the salt rotation, or the set of subprocessors changes.

- Prepared by: [name, role], [date]
- DPO review: [name], [date]
- Outcome: [proceed / proceed with conditions]
