import type { Metadata } from 'next';
import Link from 'next/link';
import { MktDocPage } from '@/components/marketing/MktDocPage';

export const metadata: Metadata = {
  title: 'Privacy',
  description: 'How Linkspire handles data: no IP logging, GDPR and CCPA tooling, and residency choice.',
  robots: { index: true, follow: true },
};

export default function PrivacyPage() {
  return (
    <MktDocPage
      title="Privacy by design"
      lead="Linkspire is built so honest analytics do not require tracking people. This page explains what we collect, what we discard, and the controls you have."
    >
      <section id="by-design">
        <h2>Core principles</h2>
        <ul>
          <li><strong>In memory first.</strong> Geo and device signals are derived from each request in memory. The IP is dropped before an analytics event is persisted.</li>
          <li><strong>No cross-day tracking.</strong> Daily unique visitors use a salt that rotates and is destroyed; we do not stitch visitors across days.</li>
          <li><strong>Data minimization.</strong> We store what you need to operate links and see aggregate performance, not identity graphs.</li>
          <li><strong>Your region, your choice.</strong> Pick US or EU residency for each workspace at sign-up.</li>
        </ul>
      </section>

      <h2>Account and workspace data</h2>
      <p>
        When you register, we store your email, password hash (Argon2id), workspace membership, and settings you configure
        (domains, links, API keys, webhooks). This is operational data required to run the service for you.
      </p>

      <h2>Redirect and analytics data</h2>
      <p>
        When someone clicks a short link, our redirect path resolves the destination and may emit an analytics event
        containing aggregated fields only (for example country, device class, browser family, referrer host, timestamp).
        Raw IPs are not written to analytics storage.
      </p>

      <h2>GDPR, CCPA, and individual rights</h2>
      <ul>
        <li>Data subject access and erasure requests (DSAR) from workspace admins</li>
        <li>Do-not-sell and Global Privacy Control (GPC) signals honored where applicable</li>
        <li>Retention schedules for raw events and aggregates documented in product settings</li>
      </ul>

      <h2>Security</h2>
      <p>
        TLS in transit, scoped API keys, role-based access, SSRF-safe URL fetching for scanners, and abuse reporting
        (`/report-abuse`) for malicious short links. Security contact and incident response align with your workspace
        region.
      </p>

      <h2>Subprocessors and transfers</h2>
      <p>
        Infrastructure is chosen to respect your selected residency. Cross-border transfers, if any, use appropriate
        safeguards (standard contractual clauses or equivalent). A detailed subprocessor list ships with enterprise
        agreements; this preview site uses the working product name Linkspire.
      </p>

      <h2>Contact</h2>
      <p>
        Privacy questions: use in-app settings under Privacy, or report abuse at{' '}
        <Link href="/report-abuse">Report abuse</Link>. For DSARs, signed-in workspace owners can initiate requests from
        the dashboard.
      </p>

      <p className="mkt-doc-cta">
        <Link href="/analytics" className="btn secondary">How analytics works</Link>
        <Link href="/register" className="btn">Get started</Link>
      </p>
    </MktDocPage>
  );
}
