'use client';
import { useActionState } from 'react';
import { bulkCreateAction, type BulkState } from '@/app/actions/links';
import { SubmitButton } from '@/components/SubmitButton';
import { Alert } from '@/components/Alert';

const initial: BulkState = {};

export function BulkCreateForm() {
  const [state, action] = useActionState(bulkCreateAction, initial);
  return (
    <form action={action} className="stack" noValidate>
      {state.error ? <Alert variant="danger">{state.error}</Alert> : null}
      {state.created != null ? (
        <Alert variant={state.failed ? 'warning' : 'success'} title={`Created ${state.created} link${state.created === 1 ? '' : 's'}${state.failed ? `, ${state.failed} failed` : ''}.`}>
          {state.errors && state.errors.length ? (
            <ul style={{ margin: '8px 0 0 18px' }}>{state.errors.map((e, i) => <li key={i}>{e}</li>)}</ul>
          ) : null}
        </Alert>
      ) : null}
      <div className="field">
        <label htmlFor="bulk-urls">Destination URLs<span className="req" aria-hidden="true">*</span></label>
        <textarea id="bulk-urls" name="urls" className="textarea" required
          placeholder={'https://example.com/one\nhttps://example.com/two'} aria-describedby="bulk-hint" />
        <span id="bulk-hint" className="hint">One http(s) URL per line, up to 100. Each is scanned before it is saved.</span>
      </div>
      <div><SubmitButton pendingLabel="Creating…">Create links</SubmitButton></div>
    </form>
  );
}
