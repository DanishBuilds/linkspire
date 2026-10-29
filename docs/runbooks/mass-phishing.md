<!-- docs/runbooks/mass-phishing.md -->
# Runbook: mass phishing / malicious link surge

**Severity:** SEV1 if the platform is actively distributing malware or phishing at scale (reputation
and user-safety risk).

## Detect

- Alerts: spike in abuse reports, a jump in `malicious`/`suspicious` scan verdicts, blocklist
  provider notices, or a sudden burst of link creation from one workspace or API key.

## Assess

- Concentrated (one workspace, key, or destination pattern) or distributed?
- Are links already live and redirecting, or caught at creation? Creation-time fast-path scanning
  rejects known-malicious destinations with a 422 and never persists a row, so most caught attempts
  never become live links.

## Mitigate

1. **Contain the source.** If one workspace or API key is responsible, revoke the key
   (`DELETE /v1/api-keys/{id}`) and suspend the workspace's write access. If one destination or
   pattern, add it to the local blocklist so the fast-path scanner rejects it at creation.
2. **Disable live malicious links.** Set affected links to a disabled status so the redirect serves
   the safety interstitial / not-found instead of forwarding. This takes effect as caches expire;
   for immediate effect, purge the affected codes from cache.
3. **Force re-scan.** Trigger the scanner to re-evaluate suspicious links ahead of the normal 24h
   re-scan cadence. Flagged links emit the `link.flagged` webhook so downstream consumers react.
4. **Refresh threat intel.** Ensure the URLHaus / Safe Browsing hash-prefix data is current.
5. Keep the hot path clean: do all of this via the management API and workers, never by adding a
   synchronous third-party call into the redirect path.

## Verify

- New malicious creations are rejected (422), live offending links no longer forward, abuse queue
  draining, verdict rates normalizing.

## Communicate and follow up

- Notify blocklist providers that offending links are disabled to protect platform reputation.
  Respond to abuse reporters as appropriate. Postmortem: detection lead time, whether limits on
  creation rate per key would have contained it sooner, and scanner coverage gaps.
