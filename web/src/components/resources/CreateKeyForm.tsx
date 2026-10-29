'use client';
import { useActionState } from 'react';
import { createKeyAction, type ResState } from '@/app/actions/resources';
import { SubmitButton } from '@/components/SubmitButton';
import { Alert } from '@/components/Alert';
import { Reveal } from '@/components/Reveal';
import { CopyButton } from '@/components/CopyButton';

const initial: ResState = {};

export function CreateKeyForm() {
  const [state, action] = useActionState(createKeyAction, initial);
  return (
    <form action={action} className="stack" noValidate>
      {state.error ? <Alert variant="danger">{state.error}</Alert> : null}
      {state.success && state.secret ? (
        <Alert variant="warning" title={state.success}>
          <div className="stack" style={{ marginTop: 8 }}>
            <Reveal value={state.secret} />
            <CopyButton value={state.secret} label="Copy key" />
          </div>
        </Alert>
      ) : null}
      <div className="field">
        <label htmlFor="key-name">Key name<span className="req" aria-hidden="true">*</span></label>
        <input id="key-name" name="name" type="text" required className="input" placeholder="e.g. CI pipeline" autoComplete="off" />
      </div>
      <div className="field">
        <label htmlFor="key-scopes">Scopes</label>
        <input id="key-scopes" name="scopes" type="text" className="input" placeholder="comma-separated, e.g. links:write,links:read" autoComplete="off" />
        <span className="hint">Optional. Leave blank for a key with the workspace default (admin) scope.</span>
      </div>
      <div><SubmitButton pendingLabel="Creating…">Create key</SubmitButton></div>
    </form>
  );
}
