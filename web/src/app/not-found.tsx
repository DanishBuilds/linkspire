import Link from 'next/link';

export default function NotFound() {
  return (
    <main id="main" className="auth-wrap">
      <div className="auth-card" style={{ textAlign: 'center' }}>
        <h1 style={{ fontSize: '2.4rem' }}>404</h1>
        <p className="muted" style={{ marginBottom: 18 }}>We couldn’t find that page.</p>
        <Link href="/dashboard" className="btn">Back to dashboard</Link>
      </div>
    </main>
  );
}
