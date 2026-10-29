'use client';
import { useActionState } from 'react';
import { registerAction, type FormState } from '@/app/actions/auth';
import { SubmitButton } from '@/components/SubmitButton';
import { Alert } from '@/components/Alert';

const initial: FormState = {};

export function RegisterForm() {
  const [state, action] = useActionState(registerAction, initial);
  return (
    <form action={action} noValidate>
      {state.error ? <div style={{ marginBottom: 16 }}><Alert variant="danger">{state.error}</Alert></div> : null}
      <div className="field">
        <label htmlFor="email">Email<span className="req" aria-hidden="true">*</span></label>
        <input id="email" name="email" type="email" autoComplete="email" required className="input" defaultValue={state.email ?? ''} />
      </div>
      <div className="field">
        <label htmlFor="password">Password<span className="req" aria-hidden="true">*</span></label>
        <input id="password" name="password" type="password" autoComplete="new-password" required minLength={10}
          className="input" aria-describedby="pw-hint" />
        <span id="pw-hint" className="hint">At least 10 characters.</span>
      </div>
      <div className="field">
        <label htmlFor="workspace">Workspace name</label>
        <input id="workspace" name="workspace" type="text" className="input" placeholder="Optional — defaults from your email" />
      </div>
      <fieldset style={{ marginBottom: 16 }}>
        <legend>Data residency</legend>
        <p className="hint" style={{ marginBottom: 8 }}>Where your workspace data is stored. This cannot be changed later.</p>
        <div className="checkbox" style={{ marginBottom: 6 }}>
          <input id="region-us" name="home_region" type="radio" value="us" defaultChecked />
          <label htmlFor="region-us" style={{ fontWeight: 400 }}>United States (us)</label>
        </div>
        <div className="checkbox">
          <input id="region-eu" name="home_region" type="radio" value="eu" />
          <label htmlFor="region-eu" style={{ fontWeight: 400 }}>European Union (eu)</label>
        </div>
      </fieldset>
      <SubmitButton pendingLabel="Creating account…" className="btn">Create account</SubmitButton>
    </form>
  );
}
