'use client';
import { useState } from 'react';
import { startAuthentication } from '@simplewebauthn/browser';
import { Alert } from '@/components/Alert';

// Passwordless sign-in with a passkey. Needs the email to locate the account's
// credentials, then runs the WebAuthn assertion and sets the session via the BFF.
export function PasskeyLogin({ next }: { next: string }) {
  const [email, setEmail] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  async function go() {
    setBusy(true); setError('');
    try {
      const beginRes = await fetch('/api/webauthn/login/begin', {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ email }),
      });
      if (!beginRes.ok) throw new Error((await beginRes.json()).error || 'No passkey for this account');
      const options = await beginRes.json();
      const assertion = await startAuthentication({ optionsJSON: options.publicKey });
      const finRes = await fetch('/api/webauthn/login/finish', {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(assertion),
      });
      if (!finRes.ok) throw new Error((await finRes.json()).error || 'Passkey sign-in failed');
      window.location.href = next.startsWith('/') ? next : '/dashboard';
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Passkey sign-in failed.');
    } finally {
      setBusy(false);
    }
  }

  return (
    <details className="stack" style={{ marginTop: 8 }}>
      <summary style={{ cursor: 'pointer', minHeight: 44, display: 'flex', alignItems: 'center', fontWeight: 600 }}>
        Sign in with a passkey
      </summary>
      <div className="stack" style={{ marginTop: 10 }}>
        {error ? <Alert variant="danger">{error}</Alert> : null}
        <div className="field" style={{ margin: 0 }}>
          <label htmlFor="pk-email">Email</label>
          <input id="pk-email" type="email" className="input" autoComplete="username webauthn"
            value={email} onChange={(e) => setEmail(e.target.value)} />
        </div>
        <div>
          <button type="button" className="btn secondary" onClick={go} disabled={busy || !email} aria-busy={busy}>
            {busy ? 'Waiting for passkey…' : 'Continue with passkey'}
          </button>
        </div>
      </div>
    </details>
  );
}
