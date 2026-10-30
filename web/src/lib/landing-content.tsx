// Static copy + data for the marketing landing page. Keeping it here keeps the
// section components lean and makes the page easy to re-word.

export const navLinks = [
  { label: 'Features', href: '/features' },
  { label: 'Analytics', href: '/analytics' },
  { label: 'Developers', href: '/developers' },
  { label: 'Privacy', href: '/privacy' },
];

export const trustItems = [
  'No credit card required',
  '10K links free',
  'SOC 2 aligned',
  'GDPR ready',
];

export const logos = [
  'Northwind', 'Vertex', 'Lumen', 'Cobalt',
  'Harbor', 'Quanta', 'Meridian', 'Atlas',
  'Nimbus', 'Forge', 'Orbit', 'Beacon',
];

export const stats = [
  { value: 50, suffix: 'M+', label: 'Links shortened' },
  { value: 2, suffix: 'B+', label: 'Redirects served' },
  { value: 99.99, suffix: '%', label: 'Uptime SLA', decimals: 2 },
  { value: 15, suffix: 'ms', label: 'Avg redirect latency' },
];

export const comparisons = [
  {
    name: 'vs Bitly',
    points: ['No link limits on the free tier', 'No visitor tracking, ever', 'Self-hostable in minutes'],
  },
  {
    name: 'vs Rebrandly',
    points: ['Open-source, no vendor lock-in', 'Privacy-first analytics', 'Honest per-day uniques'],
  },
  {
    name: 'vs Short.io',
    points: ['Faster edge redirects', 'A developer-first API', 'Signed webhooks included'],
  },
];

export const testimonials = [
  {
    featured: true,
    quote: 'We moved 40,000 links over in an afternoon. The analytics are honest, the redirects are fast, and our legal team stopped worrying about visitor tracking.',
    highlight: 'our legal team stopped worrying about visitor tracking',
    name: 'Priya Nair',
    role: 'Head of Growth, Northwind',
    initials: 'PN',
  },
  { quote: 'Finally a shortener our security team signed off on without a 3-week review.', name: 'Marco Feld', role: 'Platform Lead, Vertex', initials: 'MF' },
  { quote: 'The API is what Bitly should have shipped years ago.', name: 'Dana Osei', role: 'Staff Engineer, Lumen', initials: 'DO', tweet: true },
  { quote: 'Self-hosted it on a Friday, had custom domains live by Monday.', name: 'Sven Larsen', role: 'Founder, Cobalt', initials: 'SL' },
  { quote: 'Per-day uniques without cookies is exactly the compromise we wanted.', name: 'Aisha Rahman', role: 'Data PM, Harbor', initials: 'AR' },
];

export const pricing = [
  {
    name: 'Free',
    price: '$0',
    cadence: 'forever',
    features: ['10,000 links', '50K redirects / mo', '1 custom domain', 'Basic analytics'],
    cta: 'Start free',
  },
  {
    name: 'Pro',
    price: '$9',
    cadence: 'per month',
    popular: true,
    features: ['Unlimited links', 'Unlimited redirects', '10 custom domains', 'Full analytics', 'API access', 'QR codes'],
    cta: 'Start Pro trial',
  },
  {
    name: 'Enterprise',
    price: 'Custom',
    cadence: 'contact us',
    features: ['SSO & SCIM', 'Audit logs', 'Dedicated support', '99.99% SLA', 'On-prem deployment'],
    cta: 'Talk to sales',
  },
];

export const footerColumns: { title: string; links: [string, string][] }[] = [
  { title: 'Product', links: [['Features', '/features'], ['Analytics', '/analytics'], ['Privacy', '/privacy'], ['Pricing', '/#pricing'], ['Status', '/#']] },
  { title: 'Resources', links: [['Developers', '/developers'], ['API docs', '/developers'], ['Open source', '/#open-source'], ['Changelog', '/#']] },
  { title: 'Legal', links: [['Privacy policy', '/privacy'], ['Terms', '/#'], ['DPA', '/#'], ['Security', '/#']] },
];

export const sdkCode = `import { Linkspire } from '@linkspire/sdk';

const client = new Linkspire({ apiKey: 'ls_...' });

const link = await client.links.create({
  url: 'https://example.com/very-long-path',
  domain: 'go.yourco.com',
  tags: ['campaign-q4'],
});
// => { shortUrl: 'go.yourco.com/q4promo', clicks: 0 }`;
