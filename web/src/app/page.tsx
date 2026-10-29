import Link from 'next/link';
import type { Metadata } from 'next';
import { SiteHeader } from '@/components/marketing/SiteHeader';
import { SiteFooter } from '@/components/marketing/SiteFooter';
import { HeroPreview } from '@/components/marketing/HeroPreview';
import { Icon } from '@/components/icons';

export const metadata: Metadata = {
  title: 'Linkspire: privacy-first link management',
  description:
    'Shorten links, serve redirects at scale, and get honest analytics with no IP logging and no cross-day visitor tracking.',
  robots: { index: true, follow: true },
  openGraph: {
    title: 'Linkspire: privacy-first link management',
    description: 'Fast redirects, honest analytics, no visitor tracking.',
    type: 'website',
  },
};

const features = [
  { icon: <Icon.Shield />, title: 'Privacy-first analytics', body: 'Country, device, browser, and referrer, derived in memory. The IP is discarded before any event is stored.' },
  { icon: <Icon.Chart />, title: 'Honest metrics', body: 'Daily aggregates and per-day unique visitors. No cross-day tracking, because the salt rotates and self-destructs.' },
  { icon: <Icon.Link />, title: 'Links that do more', body: 'Custom aliases, passwords, expiry, click limits, and four redirect types. Bulk-create up to 100 at once.' },
  { icon: <Icon.Globe />, title: 'Your own domains', body: 'Bring a custom domain with guided CNAME and TXT setup. Verification and TLS are automatic once DNS is live.' },
  { icon: <Icon.Lock />, title: 'Safe by default', body: 'Every destination is scanned for malware and phishing before a link is saved. Malicious URLs never get a row.' },
  { icon: <Icon.Webhook />, title: 'Built for developers', body: 'A versioned REST API, scoped keys, idempotency, and signed webhooks for link and domain events.' },
];

export default function HomePage() {
  return (
    <div className="mkt" id="top">
      <a href="#main" className="skip-link">Skip to content</a>
      <SiteHeader />

      <main id="main">
        {/* hero */}
        <section className="hero">
          <div className="inner">
            <div className="hero-grid">
              <div>
                <span className="eyebrow"><Icon.Shield /> Privacy by design</span>
                <h1 className="hero-title">
                  Short links with <span className="grad">honest analytics</span>.
                </h1>
                <p className="hero-sub">
                  Linkspire shortens your URLs, serves redirects in milliseconds, and shows you what is working,
                  without logging a single IP address or tracking anyone across days.
                </p>
                <div className="hero-cta">
                  <Link href="/register" className="btn lg"><Icon.Plus /> Get started free</Link>
                  <Link href="/login" className="btn secondary lg">Sign in</Link>
                </div>
                <p className="hero-note">No credit card. Choose US or EU data residency at sign-up.</p>
              </div>
              <HeroPreview />
            </div>
          </div>
        </section>

        {/* trust bar */}
        <section className="trust" aria-label="Platform at a glance">
          <div className="inner">
            <div className="t"><div className="v">100M+</div><div className="k">redirects / day target</div></div>
            <div className="t"><div className="v">&lt; 30 ms</div><div className="k">redirect p99 at origin</div></div>
            <div className="t"><div className="v">RPO = 0</div><div className="k">for committed links</div></div>
            <div className="t"><div className="v">0</div><div className="k">IP addresses stored</div></div>
          </div>
        </section>

        {/* features */}
        <section className="mkt-section" id="features">
          <div className="section-head">
            <h2>Everything you need to run links at scale</h2>
            <p className="lead">A complete link platform that treats privacy as a feature, not an afterthought.</p>
          </div>
          <div className="features">
            {features.map((f) => (
              <article className="feature" key={f.title}>
                <span className="ic" aria-hidden="true">{f.icon}</span>
                <h3>{f.title}</h3>
                <p>{f.body}</p>
              </article>
            ))}
          </div>
        </section>

        {/* analytics split */}
        <section className="mkt-section" id="analytics" style={{ paddingTop: 0 }}>
          <div className="split">
            <div>
              <h2>Analytics you can trust, that your visitors can too</h2>
              <p className="lead">
                See clicks over time, top countries, devices, browsers, and referrers. All of it is aggregated,
                none of it identifies a person.
              </p>
              <ul className="checks">
                <li><span className="ok"><Icon.Check /></span><span>No raw IP addresses or full user agents are ever written to disk.</span></li>
                <li><span className="ok"><Icon.Check /></span><span>Unique visitors are counted per day with a salt that is destroyed within 48 hours.</span></li>
                <li><span className="ok"><Icon.Check /></span><span>Raw events expire after 30 days; aggregates roll up and age out on their own.</span></li>
              </ul>
            </div>
            <HeroPreview />
          </div>
        </section>

        {/* privacy band */}
        <section className="trust" id="privacy" aria-label="Privacy principles">
          <div className="inner" style={{ gridTemplateColumns: 'repeat(3, 1fr)' }}>
            <div className="t"><div className="v">In memory</div><div className="k">geo and device derived, then the IP is dropped</div></div>
            <div className="t"><div className="v">GDPR + CCPA</div><div className="k">DSARs, do-not-sell, and GPC honored</div></div>
            <div className="t"><div className="v">You choose</div><div className="k">US or EU residency per workspace</div></div>
          </div>
        </section>

        {/* developers */}
        <section className="mkt-section" id="developers">
          <div className="split">
            <div>
              <h2>Create a link in one request</h2>
              <p className="lead">
                A clean, versioned REST API with scoped keys, idempotency, and signed webhooks. Build it into your
                product, your CI, or your campaigns.
              </p>
              <div className="hero-cta">
                <Link href="/register" className="btn">Get an API key</Link>
                <Link href="/login" className="btn secondary">Open the dashboard</Link>
              </div>
            </div>
            <pre className="code-block" aria-label="Example API request">{`curl -X POST https://api.linkspire.io/v1/links \\
  -H "Authorization: Bearer lk_live_..." \\
  -H "Content-Type: application/json" \\
  -d '{
    "destination": "https://example.com/launch",
    "alias": "launch",
    "expires_at": "2026-12-31T23:59:59Z"
  }'

# 201 Created
{ "code": "launch", "short_url": "https://lnk.spr/launch" }`}</pre>
          </div>
        </section>

        {/* final CTA */}
        <section className="cta-band">
          <div className="inner">
            <h2>Start shortening links the privacy-first way</h2>
            <p className="lead" style={{ margin: '12px auto 0' }}>
              Free to start. Your data stays yours, and your visitors stay anonymous.
            </p>
            <div className="hero-cta" style={{ justifyContent: 'center', marginTop: 24 }}>
              <Link href="/register" className="btn lg"><Icon.Plus /> Create your account</Link>
              <Link href="/login" className="btn secondary lg">Sign in</Link>
            </div>
          </div>
        </section>
      </main>

      <SiteFooter />
    </div>
  );
}
