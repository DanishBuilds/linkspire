import Link from 'next/link';

// Public marketing header. Static (no client JS); nav collapses on small screens.
export function SiteHeader() {
  return (
    <header className="mkt-header">
      <div className="inner">
        <Link href="/" className="brand" aria-label="Linkspire home">
          <span className="brand-mark" aria-hidden="true" />Linkspire
        </Link>
        <nav className="mkt-nav" aria-label="Primary">
          <a href="#features">Features</a>
          <a href="#privacy">Privacy</a>
          <a href="#analytics">Analytics</a>
          <a href="#developers">Developers</a>
        </nav>
        <div className="mkt-actions">
          <Link href="/login" className="btn secondary small">Sign in</Link>
          <Link href="/register" className="btn small">Get started</Link>
        </div>
      </div>
    </header>
  );
}
