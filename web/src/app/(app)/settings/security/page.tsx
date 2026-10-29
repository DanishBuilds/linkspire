import type { Metadata } from 'next';
import * as api from '@/lib/api';
import { SecuritySetup } from '@/components/account/SecuritySetup';
import { PasskeyManager } from '@/components/account/PasskeyManager';

export const metadata: Metadata = { title: 'Security' };

export default async function SecurityPage() {
  const me = await api.me();
  return (
    <div className="stack">
      <section className="card" aria-labelledby="sec-h">
        <h2 id="sec-h" style={{ marginBottom: 14 }}>Two-factor authentication</h2>
        <SecuritySetup enabled={me.mfaEnabled} />
      </section>
      <section className="card" aria-labelledby="pk-h">
        <h2 id="pk-h" style={{ marginBottom: 14 }}>Passkeys</h2>
        <PasskeyManager />
      </section>
    </div>
  );
}
