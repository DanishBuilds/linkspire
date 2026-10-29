import Link from 'next/link';
import type { Metadata } from 'next';
import { MarketingShell } from '@/components/marketing/MarketingShell';
import { HeroPreview } from '@/components/marketing/HeroPreview';
import { Icon } from '@/components/icons';
import { apiExample, marketingFeatures } from '@/lib/marketing-content';

export const dynamic = 'force-dynamic';

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

export default function HomePage() {
  return (
    <MarketingShell>
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
          {marketingFeatures.map((f) => (
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
          <pre className="code-block" aria-label="Example API request">{apiExample}</pre>
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
    </MarketingShell>
  );
}
