import Link from 'next/link';
import type { Metadata } from 'next';
import { AbuseForm } from '@/components/abuse/AbuseForm';

export const metadata: Metadata = { title: 'Report abuse' };

export default function ReportAbusePage() {
  return (
    <>
      <a href="#main" className="skip-link">Skip to content</a>
      <main id="main" className="auth-wrap">
        <div className="auth-card" style={{ maxWidth: 560 }}>
          <div className="brand"><span className="brand-mark" aria-hidden="true" />Linkspire</div>
          <h1 style={{ marginTop: 18, marginBottom: 4 }}>Report a link</h1>
          <p className="muted" style={{ marginBottom: 20 }}>
            Found a Linkspire short link being used for phishing, malware, or other abuse? Tell us and our team will review it. No account needed.
          </p>
          <AbuseForm />
          <p className="muted" style={{ marginTop: 18 }}><Link href="/login">Back to sign in</Link></p>
        </div>
      </main>
    </>
  );
}
