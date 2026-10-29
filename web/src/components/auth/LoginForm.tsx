'use client';
import { useActionState } from 'react';
import { loginAction, type FormState } from '@/app/actions/auth';
import { SubmitButton } from '@/components/SubmitButton';
import { Alert } from '@/components/Alert';

const initial: FormState = {};

export function LoginForm({ next }: { next: string }) {
  const [state, action] = useActionState(loginAction, initial);
  return (
    <form action={action} noValidate>
      {state.error ? <div style={{ marginBottom: 16 }}><Alert variant="danger">{state.error}</Alert></div> : null}
      <input type="hidden" name="next" value={next} />
      <div className="field">
        <label htmlFor="email">Email<span className="req" aria-hidden="true">*</span></label>
        <input id="email" name="email" type="email" autoComplete="email" required
          className="input" defaultValue={state.email ?? ''} aria-invalid={state.error ? true : undefined} />
      </div>
      <div className="field">
        <label htmlFor="password">Password<span className="req" aria-hidden="true">*</span></label>
        <input id="password" name="password" type="password" autoComplete="current-password" required className="input" />
      </div>
      <div className="field">
        <label htmlFor="totp">Authentication code</label>
        <input id="totp" name="totp" inputMode="numeric" pattern="[0-9]*" autoComplete="one-time-code"
          className="input" placeholder="6-digit code (if 2FA is on)"
          aria-describedby="totp-hint" defaultValue="" autoFocus={state.needsTotp} />
        <span id="totp-hint" className="hint">Only required if you enabled two-factor authentication.</span>
      </div>
      <SubmitButton pendingLabel="Signing in…" className="btn" >Sign in</SubmitButton>
    </form>
  );
}
