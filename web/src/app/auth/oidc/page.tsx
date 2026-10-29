'use client';
import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { Alert } from '@/components/Alert';

// Reads the JWT from the URL fragment (never sent to a server), exchanges it for
// an httpOnly session cookie, then lands on the dashboard.
export default function OidcLanding() {
  const router = useRouter();
  const [error, setError] = useState('');

  useEffect(() => {
    const hash = window.location.hash.replace(/^#/, '');
    const token = new URLSearchParams(hash).get('token');
    if (!token) {
      setError('No sign-in token was returned. Please try again.');
      return;
    }
    (async () => {
      try {
        const res = await fetch('/auth/oidc/complete', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ token }),
        });
        if (!res.ok) throw new Error('exchange failed');
        window.location.hash = '';
        router.replace('/dashboard');
      } catch {
        setError('We could not complete sign-in. Please try again.');
      }
    })();
  }, [router]);

  return (
    <div className="auth-card">
      <div className="brand"><span className="brand-mark" aria-hidden="true" />Linkspire</div>
      <h1 style={{ marginTop: 18, marginBottom: 12 }}>Signing you in…</h1>
      {error ? (
        <>
          <Alert variant="danger">{error}</Alert>
          <p style={{ marginTop: 16 }}><Link href="/login">Back to sign in</Link></p>
        </>
      ) : (
        <p className="muted" aria-live="polite">Completing single sign-on, one moment.</p>
      )}
    </div>
  );
}
