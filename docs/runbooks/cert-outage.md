<!-- docs/runbooks/cert-outage.md -->
# Runbook: TLS certificate outage

**Severity:** SEV1 if platform or many custom domains cannot serve HTTPS; SEV2 for a single custom
domain.

## Background

Linkspire issues certificates on demand. The redirect edge asks an internal authority endpoint,
`GET /internal/tls/ask?domain=...` with an `X-Ask-Token`, which returns 200 **only** for domains in
`state=active`. Pending or unverified domains return 404 so ACME issuance cannot be abused
(ADR-006). Certificates are obtained via ACME and cached at the edge.

## Detect

- Alerts: TLS handshake failures, ACME issuance or renewal errors, a spike in `tls/ask` 404s or
  5xxs, certificates nearing expiry.

## Assess

- Scope: the platform domain (`lnk.linkspire.io`) or customer custom domains?
- Cause: ACME provider problem, rate limiting, the ask endpoint rejecting a domain that should be
  active, DNS change on a custom domain, or an expired cached cert.

## Mitigate

1. **Ask endpoint rejecting an active domain:** confirm the domain's `state` is `active` in the DB
   and that the `X-Ask-Token` the edge presents matches `TLS_ASK_TOKEN`. A newly verified domain
   must be `active`, not merely `verified`, before issuance is authorized.
2. **ACME rate limit or provider outage:** back off and rely on cached certificates (they are valid
   until expiry). If a configured backup ACME provider exists, switch to it. Do not loosen the ask
   endpoint to issue for unverified hosts.
3. **Custom domain DNS changed:** the domain verifier will move it out of `active`; this is correct.
   Ask the customer to restore the CNAME/TXT records; re-verification restores issuance.
4. **Platform cert expired:** this should never happen with renewal; if it did, trigger manual
   issuance for the platform domain and investigate why auto-renewal lapsed.

## Verify

- Fresh handshake succeeds (`openssl s_client -connect <host>:443` shows a valid, not-expired
  chain), issuance/renewal metrics healthy, 404 rate on `tls/ask` back to only genuinely inactive
  hosts.

## Recover and follow up

- Confirm all near-expiry certs renewed. Postmortem: renewal lead time, provider redundancy, and
  monitoring of certificate expiry windows.
