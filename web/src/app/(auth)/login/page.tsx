import Link from 'next/link';
import type { Metadata } from 'next';
import { LoginForm } from '@/components/auth/LoginForm';

export const metadata: Metadata = { title: 'Sign in' };

export default async function LoginPage({ searchParams }: { searchParams: Promise<{ next?: string }> }) {
  const sp = await searchParams;
  const next = sp.next && sp.next.startsWith('/') ? sp.next : '/dashboard';
  return (
    <div className="auth-card">
      <div className="brand"><span className="brand-mark" aria-hidden="true" />Linkspire</div>
      <h1 style={{ marginTop: 18, marginBottom: 4 }}>Sign in</h1>
      <p className="muted" style={{ marginBottom: 20 }}>Access your link workspace.</p>
      <LoginForm next={next} />
      <p className="muted" style={{ marginTop: 18 }}>
        No account? <Link href="/register">Create one</Link>
      </p>
    </div>
  );
}
