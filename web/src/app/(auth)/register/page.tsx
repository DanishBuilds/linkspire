import Link from 'next/link';
import type { Metadata } from 'next';
import { RegisterForm } from '@/components/auth/RegisterForm';

export const metadata: Metadata = { title: 'Create account' };

export default function RegisterPage() {
  return (
    <div className="auth-card">
      <div className="brand"><span className="brand-mark" aria-hidden="true" />Linkspire</div>
      <h1 style={{ marginTop: 18, marginBottom: 4 }}>Create your account</h1>
      <p className="muted" style={{ marginBottom: 20 }}>Start shortening links in a privacy-first workspace.</p>
      <RegisterForm />
      <p className="muted" style={{ marginTop: 18 }}>
        Already have an account? <Link href="/login">Sign in</Link>
      </p>
    </div>
  );
}
