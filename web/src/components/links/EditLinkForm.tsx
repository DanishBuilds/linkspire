'use client';
import { useActionState } from 'react';
import { patchLinkAction, type LinkFormState } from '@/app/actions/links';
import { SubmitButton } from '@/components/SubmitButton';
import { Alert } from '@/components/Alert';
import type { Link } from '@/lib/types';

const initial: LinkFormState = {};

export function EditLinkForm({ link }: { link: Link }) {
  const [state, action] = useActionState(patchLinkAction, initial);
  return (
    <form action={action} className="stack" noValidate>
      {state.error ? <Alert variant="danger">{state.error}</Alert> : null}
      {state.success ? <Alert variant="success">{state.success}</Alert> : null}
      <input type="hidden" name="domain_id" value={link.domainId} />
      <input type="hidden" name="code" value={link.code} />
      <div className="field">
        <label htmlFor="e-dest">Destination URL</label>
        <input id="e-dest" name="destination" type="url" className="input" defaultValue={link.destination} />
      </div>
      <div className="grid cols-2">
        <div className="field">
          <label htmlFor="e-status">Status</label>
          <select id="e-status" name="status" className="select" defaultValue={link.status}>
            <option value="active">active</option>
            <option value="disabled">disabled</option>
            {link.status === 'password_protected' ? <option value="password_protected">password protected</option> : null}
          </select>
        </div>
        <div className="field">
          <label htmlFor="e-rt">Redirect type</label>
          <select id="e-rt" name="redirect_type" className="select" defaultValue={String(link.redirectType)}>
            <option value="302">302 — Found</option>
            <option value="301">301 — Moved Permanently</option>
            <option value="307">307 — Temporary Redirect</option>
            <option value="308">308 — Permanent Redirect</option>
          </select>
        </div>
      </div>
      <div><SubmitButton pendingLabel="Saving…">Save changes</SubmitButton></div>
    </form>
  );
}
