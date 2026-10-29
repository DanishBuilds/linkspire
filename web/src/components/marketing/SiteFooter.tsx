import Link from 'next/link';

export function SiteFooter() {
  const year = new Date().getFullYear();
  return (
    <footer className="mkt-footer">
      <div className="inner">
        <div className="footer-cols">
          <div>
            <Link href="/" className="brand" aria-label="Linkspire home">
              <span className="brand-mark" aria-hidden="true" />Linkspire
            </Link>
            <p className="muted" style={{ marginTop: 10, maxWidth: '34ch' }}>
              The privacy-first link platform. Fast redirects, honest analytics, no visitor tracking.
            </p>
          </div>
          <div>
            <h4>Product</h4>
            <ul>
              <li><Link href="/features">Features</Link></li>
              <li><Link href="/analytics">Analytics</Link></li>
              <li><Link href="/privacy">Privacy by design</Link></li>
              <li><Link href="/register">Get started</Link></li>
            </ul>
          </div>
          <div>
            <h4>Developers</h4>
            <ul>
              <li><Link href="/developers">API and webhooks</Link></li>
              <li><Link href="/login">Sign in</Link></li>
              <li><Link href="/report-abuse">Report abuse</Link></li>
            </ul>
          </div>
          <div>
            <h4>Company</h4>
            <ul>
              <li><Link href="/privacy">Privacy</Link></li>
              <li><Link href="/">Home</Link></li>
            </ul>
          </div>
        </div>
        <div className="footer-bottom">
          <span>© {year} Linkspire. Working name.</span>
          <span>No IP logging. No cross-day tracking.</span>
        </div>
      </div>
    </footer>
  );
}
