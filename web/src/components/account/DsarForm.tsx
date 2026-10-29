'use client';
import { useActionState } from 'react';
import { createDsarAction, type AccountState } from '@/app/actions/account';
import { SubmitButton } from '@/components/SubmitButton';
import { Alert } from '@/components/Alert';

const initial: AccountState = {};

export function DsarForm() {
  const [state, action] = useActionState(createDsarAction, initial);
  return (
    <form action={action} className="stack" noValidate>
      {state.error ? <Alert variant="danger">{state.error}</Alert> : null}
      {state.success ? <Alert variant="success">{state.success}</Alert> : null}
      <div className="grid cols-2">
        <div className="field">
          <label htmlFor="dsar-type">Request type<span className="req" aria-hidden="true">*</span></label>
          <select id="dsar-type" name="type" className="select" defaultValue="access" required>
            <option value="access">Access my data</option>
            <option value="rectification">Rectify my data</option>
            <option value="erasure">Erase my data</option>
            <option value="restriction">Restrict processing</option>
            <option value="objection">Object to processing</option>
            <option value="do_not_sell">Do not sell or share</option>
          </select>
        </div>
        <div className="field">
          <label htmlFor="dsar-regime">Regime</label>
          <select id="dsar-regime" name="regime" className="select" defaultValue="gdpr">
            <option value="gdpr">GDPR (30-day response)</option>
            <option value="ccpa">CCPA/CPRA (45-day response)</option>
          </select>
        </div>
      </div>
      <div><SubmitButton pendingLabel="Filing…">File request</SubmitButton></div>
    </form>
  );
}
