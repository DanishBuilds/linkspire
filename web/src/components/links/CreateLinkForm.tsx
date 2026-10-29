'use client';
import { useActionState } from 'react';
import { createLinkAction, type LinkFormState } from '@/app/actions/links';
import { SubmitButton } from '@/components/SubmitButton';
import { Alert } from '@/components/Alert';
import { CopyButton } from '@/components/CopyButton';
import type { DomainRecord } from '@/lib/types';

const initial: LinkFormState = {};

export function CreateLinkForm({ domains, canWrite }: { domains: DomainRecord[]; canWrite: boolean }) {
  const [state, action] = useActionState(createLinkAction, initial);
  const usable = domains.filter((d) => d.state === 'active');

  if (!canWrite) {
    return <Alert variant="info" title="Read-only access">Your role can view links but not create them. Ask a workspace admin for member access.</Alert>;
  }

  return (
    <form action={action} className="stack" noValidate>
      {state.error ? <Alert variant="danger">{state.error}</Alert> : null}
      {state.success ? (
        <Alert variant="success" title={state.success}>
          {state.shortUrl ? (
            <div className="copy-row" style={{ marginTop: 8 }}>
              <code className="code-block">{state.shortUrl}</code>
              <CopyButton value={state.shortUrl} label="Copy link" />
            </div>
          ) : null}
        </Alert>
      ) : null}

      <div className="field">
        <label htmlFor="destination">Destination URL<span className="req" aria-hidden="true">*</span></label>
        <input id="destination" name="destination" type="url" required className="input"
          placeholder="https://example.com/page" autoComplete="off" />
        <span className="hint">Must be http(s). Malicious destinations are rejected automatically.</span>
      </div>

      <div className="grid cols-2">
        <div className="field">
          <label htmlFor="domain_id">Domain</label>
          <select id="domain_id" name="domain_id" className="select" defaultValue="">
            <option value="">Default ({process.env.NEXT_PUBLIC_SHORT_BASE_URL ? new URL(process.env.NEXT_PUBLIC_SHORT_BASE_URL).host : 'platform domain'})</option>
            {usable.map((d) => <option key={d.id} value={d.id}>{d.hostname}</option>)}
          </select>
        </div>
        <div className="field">
          <label htmlFor="redirect_type">Redirect type</label>
          <select id="redirect_type" name="redirect_type" className="select" defaultValue="302">
            <option value="302">302 — Found (temporary, default)</option>
            <option value="301">301 — Moved Permanently</option>
            <option value="307">307 — Temporary Redirect</option>
            <option value="308">308 — Permanent Redirect</option>
          </select>
        </div>
      </div>

      <div className="field">
        <label htmlFor="alias">Custom alias</label>
        <input id="alias" name="alias" type="text" className="input" minLength={3} maxLength={64}
          placeholder="Optional — 3–64 chars; leave blank for an auto code" autoComplete="off" />
      </div>

      <details>
        <summary style={{ cursor: 'pointer', fontWeight: 600, minHeight: 44, display: 'flex', alignItems: 'center' }}>
          Advanced options
        </summary>
        <div className="stack" style={{ marginTop: 12 }}>
          <div className="grid cols-2">
            <div className="field">
              <label htmlFor="password">Password protection</label>
              <input id="password" name="password" type="password" className="input" autoComplete="new-password"
                placeholder="Optional passphrase" />
              <span className="hint">Visitors must enter this before the redirect.</span>
            </div>
            <div className="field">
              <label htmlFor="click_limit">Click limit</label>
              <input id="click_limit" name="click_limit" type="number" min={1} className="input" placeholder="Optional" />
            </div>
          </div>
          <div className="field">
            <label htmlFor="expires_at">Expires at (UTC)</label>
            <input id="expires_at" name="expires_at" type="datetime-local" className="input" />
          </div>
          <div className="field">
            <label htmlFor="title">Title</label>
            <input id="title" name="title" type="text" className="input" placeholder="Optional — overrides scraped metadata" />
          </div>
        </div>
      </details>

      <div><SubmitButton pendingLabel="Creating…">Create link</SubmitButton></div>
    </form>
  );
}
