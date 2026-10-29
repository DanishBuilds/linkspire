'use client';
import { useActionState } from 'react';
import { createWebhookAction, type ResState } from '@/app/actions/resources';
import { SubmitButton } from '@/components/SubmitButton';
import { Alert } from '@/components/Alert';
import { Reveal } from '@/components/Reveal';
import { CopyButton } from '@/components/CopyButton';

const initial: ResState = {};
const EVENTS = ['link.created', 'link.flagged', 'domain.verified'];

export function CreateWebhookForm() {
  const [state, action] = useActionState(createWebhookAction, initial);
  return (
    <form action={action} className="stack" noValidate>
      {state.error ? <Alert variant="danger">{state.error}</Alert> : null}
      {state.success && state.secret ? (
        <Alert variant="warning" title={state.success}>
          <div className="stack" style={{ marginTop: 8 }}>
            <p className="hint">Signing secret (HMAC-SHA256). Verify the <code>X-Linkspire-Signature</code> header with it.</p>
            <Reveal value={state.secret} />
            <CopyButton value={state.secret} label="Copy secret" />
          </div>
        </Alert>
      ) : null}
      <div className="field">
        <label htmlFor="wh-url">Endpoint URL<span className="req" aria-hidden="true">*</span></label>
        <input id="wh-url" name="url" type="url" required className="input" placeholder="https://example.com/hooks/linkspire" autoComplete="off" />
        <span className="hint">Must be HTTPS. Deliveries are signed and retried with backoff.</span>
      </div>
      <fieldset>
        <legend>Events<span className="req" aria-hidden="true">*</span></legend>
        <input type="hidden" name="events" id="events-hidden" />
        <EventChecks />
      </fieldset>
      <div><SubmitButton pendingLabel="Registering…">Register webhook</SubmitButton></div>
    </form>
  );
}

// Keeps a comma-joined hidden input in sync so the server action gets a simple string.
function EventChecks() {
  function sync() {
    const boxes = Array.from(document.querySelectorAll<HTMLInputElement>('input[data-event]'));
    const hidden = document.getElementById('events-hidden') as HTMLInputElement | null;
    if (hidden) hidden.value = boxes.filter((b) => b.checked).map((b) => b.value).join(',');
  }
  return (
    <div className="stack" style={{ marginTop: 8 }}>
      {EVENTS.map((ev) => (
        <div className="checkbox" key={ev}>
          <input id={`ev-${ev}`} type="checkbox" data-event value={ev} onChange={sync} />
          <label htmlFor={`ev-${ev}`} style={{ fontWeight: 400 }}><code>{ev}</code></label>
        </div>
      ))}
    </div>
  );
}
