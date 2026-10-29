'use client';
import { useActionState } from 'react';
import { createDomainAction, type ResState } from '@/app/actions/resources';
import { SubmitButton } from '@/components/SubmitButton';
import { Alert } from '@/components/Alert';

const initial: ResState = {};

export function AddDomainForm() {
  const [state, action] = useActionState(createDomainAction, initial);
  return (
    <form action={action} className="stack" noValidate>
      {state.error ? <Alert variant="danger">{state.error}</Alert> : null}
      {state.success ? <Alert variant="success">{state.success}</Alert> : null}
      <div className="field">
        <label htmlFor="hostname">Hostname<span className="req" aria-hidden="true">*</span></label>
        <input id="hostname" name="hostname" type="text" required className="input"
          placeholder="go.example.com" autoComplete="off" spellCheck={false} />
        <span className="hint">The domain or subdomain you want short links served from.</span>
      </div>
      <div className="checkbox">
        <input id="apex" name="apex" type="checkbox" />
        <label htmlFor="apex" style={{ fontWeight: 400 }}>This is an apex (root) domain, e.g. example.com</label>
      </div>
      <div><SubmitButton pendingLabel="Adding…">Add domain</SubmitButton></div>
    </form>
  );
}
