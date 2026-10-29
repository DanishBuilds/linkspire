'use client';
import { useActionState } from 'react';
import { requestQrAction, type LinkFormState } from '@/app/actions/links';
import { SubmitButton } from '@/components/SubmitButton';
import { Alert } from '@/components/Alert';
import type { Link } from '@/lib/types';

const initial: LinkFormState = {};

export function QrForm({ link }: { link: Link }) {
  const [state, action] = useActionState(requestQrAction, initial);
  return (
    <form action={action} className="stack" noValidate>
      {state.error ? <Alert variant="danger">{state.error}</Alert> : null}
      {state.success ? <Alert variant="success">{state.success}</Alert> : null}
      <input type="hidden" name="domain_id" value={link.domainId} />
      <input type="hidden" name="code" value={link.code} />
      <fieldset>
        <legend>Format</legend>
        <div className="cluster" style={{ marginTop: 8 }}>
          <div className="checkbox"><input id="qr-png" type="radio" name="format" value="png" defaultChecked /><label htmlFor="qr-png" style={{ fontWeight: 400 }}>PNG</label></div>
          <div className="checkbox"><input id="qr-svg" type="radio" name="format" value="svg" /><label htmlFor="qr-svg" style={{ fontWeight: 400 }}>SVG</label></div>
        </div>
      </fieldset>
      <div><SubmitButton className="btn secondary" pendingLabel="Queuing…">Generate QR code</SubmitButton></div>
    </form>
  );
}
