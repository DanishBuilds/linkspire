'use client';
import { useActionState } from 'react';
import { doNotSellAction, type AccountState } from '@/app/actions/account';
import { SubmitButton } from '@/components/SubmitButton';
import { Alert } from '@/components/Alert';

const initial: AccountState = {};

export function DoNotSellForm({ alreadySet }: { alreadySet: boolean }) {
  const [state, action] = useActionState(doNotSellAction, initial);
  if (alreadySet || state.success) {
    return <Alert variant="success" title="Preference recorded">Your Do Not Sell or Share preference is on. Linkspire does not sell or share personal information.</Alert>;
  }
  return (
    <form action={action} className="stack">
      {state.error ? <Alert variant="danger">{state.error}</Alert> : null}
      <p className="muted">Record your CCPA/CPRA “Do Not Sell or Share” preference. Linkspire does not sell personal information; this sets the flag on your account.</p>
      <div><SubmitButton className="btn secondary" pendingLabel="Saving…">Record my preference</SubmitButton></div>
    </form>
  );
}
