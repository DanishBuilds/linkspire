'use client';
import Link from 'next/link';
import { useActionState } from 'react';
import { createLinkAction, type LinkFormState } from '@/app/actions/links';
import { SubmitButton } from '@/components/SubmitButton';
import { Alert } from '@/components/Alert';
import { CopyButton } from '@/components/CopyButton';
import { Icon } from '@/components/icons';
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
            <div className="stack" style={{ marginTop: 10, gap: 12 }}>
              <div className="copy-row">
                <code className="code-block">{state.shortUrl}</code>
                <CopyButton value={state.shortUrl} label="Copy link" />
              </div>
              {state.code != null && state.domainId != null ? (
                <div className="cluster">
                  <Link
                    href={`/links/${state.domainId}/${encodeURIComponent(state.code)}#qr-code`}
                    className="btn"
                  >
                    <Icon.Qr /> Create QR code for this link
                  </Link>
                  <Link
                    href={`/links/${state.domainId}/${encodeURIComponent(state.code)}`}
                    className="btn secondary"
                  >
                    View link details
                  </Link>
                </div>
              ) : null}
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

      <details className="disclosure">
        <summary>
          <span className="disclosure-main">
            <span className="disclosure-ic" aria-hidden="true"><Icon.Settings /></span>
            <span>
              <span className="disclosure-title">More options</span>
              <span className="disclosure-sub">Password, expiry date, click limit, custom title (all optional)</span>
            </span>
          </span>
          <svg className="chev" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="m6 9 6 6 6-6" /></svg>
        </summary>
        <div className="disclosure-body stack">
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
