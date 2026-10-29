'use client';
import { useActionState } from 'react';
import { reportAbuseAction, type AbuseState } from '@/app/actions/abuse';
import { SubmitButton } from '@/components/SubmitButton';
import { Alert } from '@/components/Alert';

const initial: AbuseState = {};

export function AbuseForm() {
  const [state, action] = useActionState(reportAbuseAction, initial);
  if (state.success) return <Alert variant="success" title="Report received">{state.success}</Alert>;
  return (
    <form action={action} className="stack" noValidate>
      {state.error ? <Alert variant="danger">{state.error}</Alert> : null}
      <div className="field">
        <label htmlFor="short_url">Short link<span className="req" aria-hidden="true">*</span></label>
        <input id="short_url" name="short_url" type="text" required className="input" placeholder="https://lnk.linkspire.io/abc123" autoComplete="off" />
      </div>
      <div className="field">
        <label htmlFor="reason">What’s wrong with it?<span className="req" aria-hidden="true">*</span></label>
        <textarea id="reason" name="reason" required minLength={10} className="textarea"
          placeholder="Describe the phishing, malware, or other abuse you’ve observed." />
      </div>
      <div><SubmitButton pendingLabel="Submitting…">Submit report</SubmitButton></div>
    </form>
  );
}
