import { Icon } from '@/components/icons';

export const marketingFeatures = [
  {
    icon: <Icon.Shield />,
    title: 'Privacy-first analytics',
    body: 'Country, device, browser, and referrer, derived in memory. The IP is discarded before any event is stored.',
  },
  {
    icon: <Icon.Chart />,
    title: 'Honest metrics',
    body: 'Daily aggregates and per-day unique visitors. No cross-day tracking, because the salt rotates and self-destructs.',
  },
  {
    icon: <Icon.Link />,
    title: 'Links that do more',
    body: 'Custom aliases, passwords, expiry, click limits, and four redirect types. Bulk-create up to 100 at once.',
  },
  {
    icon: <Icon.Globe />,
    title: 'Your own domains',
    body: 'Bring a custom domain with guided CNAME and TXT setup. Verification and TLS are automatic once DNS is live.',
  },
  {
    icon: <Icon.Lock />,
    title: 'Safe by default',
    body: 'Every destination is scanned for malware and phishing before a link is saved. Malicious URLs never get a row.',
  },
  {
    icon: <Icon.Webhook />,
    title: 'Built for developers',
    body: 'A versioned REST API, scoped keys, idempotency, and signed webhooks for link and domain events.',
  },
] as const;

export const apiExample = `curl -X POST https://api.linkspire.io/v1/links \\
  -H "Authorization: Bearer lk_live_..." \\
  -H "Content-Type: application/json" \\
  -d '{
    "destination": "https://example.com/launch",
    "alias": "launch",
    "expires_at": "2026-12-31T23:59:59Z"
  }'

# 201 Created
{ "code": "launch", "short_url": "https://lnk.spr/launch" }`;
