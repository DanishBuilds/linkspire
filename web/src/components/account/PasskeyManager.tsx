'use client';
import { useCallback, useEffect, useState } from 'react';
import { startRegistration } from '@simplewebauthn/browser';
import { Alert } from '@/components/Alert';
import type { Passkey } from '@/lib/types';

// Passkey (WebAuthn) registration and management. Talks to the BFF routes under
// /api/webauthn, which attach the session token server-side.
export function PasskeyManager() {
  const [keys, setKeys] = useState<Passkey[]>([]);
  const [msg, setMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const r = await fetch('/api/webauthn/credentials');
      if (r.ok) setKeys((await r.json()).data ?? []);
    } catch { /* ignore */ }
  }, []);
  useEffect(() => { void load(); }, [load]);

  async function add() {
    setBusy(true); setMsg(null);
    try {
      const label = window.prompt('Name this passkey', 'My device') || 'Passkey';
      const beginRes = await fetch('/api/webauthn/register/begin', { method: 'POST' });
      if (!beginRes.ok) throw new Error((await beginRes.json()).error || 'Could not start registration');
      const options = await beginRes.json();
      const attestation = await startRegistration({ optionsJSON: options.publicKey });
      const finRes = await fetch(`/api/webauthn/register/finish?label=${encodeURIComponent(label)}`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(attestation),
      });
      if (!finRes.ok) throw new Error((await finRes.json()).error || 'Could not finish registration');
      setMsg({ ok: true, text: 'Passkey added.' });
      await load();
    } catch (e) {
      setMsg({ ok: false, text: e instanceof Error ? e.message : 'Could not add passkey.' });
    } finally {
      setBusy(false);
    }
  }

  async function remove(id: number) {
    if (!window.confirm('Remove this passkey?')) return;
    await fetch(`/api/webauthn/credentials/${id}`, { method: 'DELETE' });
    await load();
  }

  return (
    <div className="stack">
      {msg ? <Alert variant={msg.ok ? 'success' : 'danger'}>{msg.text}</Alert> : null}
      <p className="muted">Sign in with your device biometrics or a security key instead of a code.</p>
      {keys.length > 0 ? (
        <ul style={{ listStyle: 'none', padding: 0, margin: 0, display: 'flex', flexDirection: 'column', gap: 8 }}>
          {keys.map((k) => (
            <li key={k.id} className="spread" style={{ border: '1px solid var(--border)', borderRadius: 8, padding: '10px 12px' }}>
              <span>{k.label || 'Passkey'}</span>
              <button type="button" className="btn secondary small" onClick={() => remove(k.id)}>Remove</button>
            </li>
          ))}
        </ul>
      ) : (
        <p className="muted">No passkeys yet.</p>
      )}
      <div>
        <button type="button" className="btn" onClick={add} disabled={busy} aria-busy={busy}>
          {busy ? 'Working…' : 'Add a passkey'}
        </button>
      </div>
    </div>
  );
}
