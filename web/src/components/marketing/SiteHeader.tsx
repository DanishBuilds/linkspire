import Link from 'next/link';
import { getToken } from '@/lib/session';

// Public marketing header. Server component: shows a Dashboard link when the
// visitor already has a session, and a zero-JS <details> menu on small screens.
export async function SiteHeader() {
  const loggedIn = Boolean(await getToken());

  const links = (
    <>
      <a href="#features">Features</a>
      <a href="#privacy">Privacy</a>
      <a href="#analytics">Analytics</a>
      <a href="#developers">Developers</a>
    </>
  );
  const actions = loggedIn ? (
    <Link href="/dashboard" className="btn small">Go to dashboard</Link>
  ) : (
    <>
      <Link href="/login" className="btn secondary small">Sign in</Link>
      <Link href="/register" className="btn small">Get started</Link>
    </>
  );

  return (
    <header className="mkt-header">
      <div className="inner">
        <Link href="/" className="brand" aria-label="Linkspire home">
          <span className="brand-mark" aria-hidden="true" />Linkspire
        </Link>
        <nav className="mkt-nav" aria-label="Primary">{links}</nav>
        <div className="mkt-actions">{actions}</div>

        <details className="mkt-menu">
          <summary className="btn secondary small" aria-label="Open menu">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true" focusable="false">
              <path d="M4 7h16M4 12h16M4 17h16" />
            </svg>
            Menu
          </summary>
          <div className="mkt-menu-panel">
            <nav aria-label="Mobile navigation">{links}</nav>
            <div className="mkt-menu-actions">{actions}</div>
          </div>
        </details>
      </div>
    </header>
  );
}
