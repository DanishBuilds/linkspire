import type { Metadata } from 'next';
import Link from 'next/link';
import { MktDocPage } from '@/components/marketing/MktDocPage';
import { HeroPreview } from '@/components/marketing/HeroPreview';
import { Icon } from '@/components/icons';

export const metadata: Metadata = {
  title: 'Analytics',
  description: 'Privacy-safe click analytics: aggregates, daily uniques, and no IP storage.',
  robots: { index: true, follow: true },
};

export default function AnalyticsPage() {
  return (
    <MktDocPage
      title="Analytics you can trust, that your visitors can too"
      lead="See clicks over time, top countries, devices, browsers, and referrers. All of it is aggregated; none of it identifies a person."
    >
      <div className="split" style={{ marginBottom: 32 }}>
        <ul className="checks">
          <li><span className="ok"><Icon.Check /></span><span>No raw IP addresses or full user agents are ever written to disk.</span></li>
          <li><span className="ok"><Icon.Check /></span><span>Unique visitors are counted per day with a salt that is destroyed within 48 hours.</span></li>
          <li><span className="ok"><Icon.Check /></span><span>Raw events expire after 30 days; aggregates roll up and age out on their own.</span></li>
        </ul>
        <HeroPreview />
      </div>

      <h2>What we measure</h2>
      <ul>
        <li>Total clicks and clicks over time (hourly and daily rollups)</li>
        <li>Approximate country and region from in-memory geo lookup</li>
        <li>Device class and browser family parsed from the user agent, then discarded</li>
        <li>Referrer hostname when present (never full query strings with PII)</li>
        <li>Per-day unique visitors using a salted hash that cannot be reversed</li>
      </ul>

      <h2>What we never store</h2>
      <ul>
        <li>Raw IP addresses or persistent visitor IDs</li>
        <li>Cross-day profiles or fingerprinting signals</li>
        <li>Full user-agent strings in the analytics store</li>
      </ul>

      <h2>Retention</h2>
      <p>
        Raw click events are kept for up to 30 days, then deleted. Daily aggregates remain for reporting windows defined
        by your workspace policy. Unique-visitor salts rotate and are destroyed within 48 hours so counts cannot be
        joined across days.
      </p>

      <h2>Dashboard views</h2>
      <p>
        Signed-in users see per-link trends, top geographies, devices, browsers, and referrers in the control plane.
        Exports and API access follow the same aggregation rules; there is no “download all IPs” path.
      </p>

      <p className="mkt-doc-cta">
        <Link href="/privacy" className="btn secondary">Privacy by design</Link>
        <Link href="/register" className="btn">Create an account</Link>
      </p>
    </MktDocPage>
  );
}
