import type { Metadata } from 'next';
import Link from 'next/link';
import { MktDocPage } from '@/components/marketing/MktDocPage';
import { marketingFeatures } from '@/lib/marketing-content';

export const metadata: Metadata = {
  title: 'Features',
  description: 'Link management, custom domains, safety scanning, and privacy-first analytics at scale.',
  robots: { index: true, follow: true },
};

export default function FeaturesPage() {
  return (
    <MktDocPage
      title="Everything you need to run links at scale"
      lead="A complete link platform that treats privacy as a feature, not an afterthought."
    >
      <div className="features" style={{ marginTop: 8 }}>
        {marketingFeatures.map((f) => (
          <article className="feature" key={f.title}>
            <span className="ic" aria-hidden="true">{f.icon}</span>
            <h3>{f.title}</h3>
            <p>{f.body}</p>
          </article>
        ))}
      </div>

      <h2>Link controls</h2>
      <p>
        Every link supports custom aliases, optional passwords, scheduled expiry, click limits, and four redirect types
        (temporary 302, permanent 301, masked, and interstitial warning pages). Bulk-create up to 100 links in one
        request for campaigns and migrations.
      </p>

      <h2>Custom domains</h2>
      <p>
        Attach your own hostname with guided DNS (CNAME and TXT verification). Once DNS is live, TLS certificates are
        issued automatically and redirects serve from your brand, not ours.
      </p>

      <h2>Safety before save</h2>
      <p>
        Destinations are scanned for malware and phishing before a row is committed. Suspicious URLs are blocked;
        malicious URLs never receive a short code.
      </p>

      <h2>Workspace-ready</h2>
      <ul>
        <li>Role-based access for owners, admins, members, and viewers</li>
        <li>US or EU data residency chosen at workspace creation</li>
        <li>API keys with scopes, idempotency keys, and signed webhooks</li>
      </ul>

      <p className="mkt-doc-cta">
        <Link href="/register" className="btn">Get started free</Link>
        <Link href="/developers" className="btn secondary">Developer docs</Link>
      </p>
    </MktDocPage>
  );
}
