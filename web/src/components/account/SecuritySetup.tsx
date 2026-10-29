'use client';
import Image from 'next/image';
import { useActionState } from 'react';
import { startMfaSetupAction, enableMfaAction, type MfaSetupState, type AccountState } from '@/app/actions/account';
import { SubmitButton } from '@/components/SubmitButton';
import { Alert } from '@/components/Alert';
import { CopyButton } from '@/components/CopyButton';

const s0: MfaSetupState = {};
const e0: AccountState = {};

export function SecuritySetup({ enabled }: { enabled: boolean }) {
  const [setup, startAction] = useActionState(startMfaSetupAction, s0);
  const [enableState, enableAction] = useActionState(enableMfaAction, e0);

  if (enabled || enableState.success) {
    return <Alert variant="success" title="Two-factor authentication is enabled">Your account requires an authenticator code at sign-in.</Alert>;
  }

  return (
    <div className="stack">
      {setup.error ? <Alert variant="danger">{setup.error}</Alert> : null}
      {!setup.secret ? (
        <form action={startAction}>
          <p className="muted" style={{ marginBottom: 12 }}>Protect your account with a time-based one-time code (TOTP) from an authenticator app.</p>
          <SubmitButton pendingLabel="Preparing…" className="btn">Set up two-factor authentication</SubmitButton>
        </form>
      ) : (
        <div className="stack">
          <p>1. Scan this QR code with your authenticator app, or enter the secret manually.</p>
          {setup.qrDataUrl ? (
            <Image src={setup.qrDataUrl} alt="QR code to add Linkspire to your authenticator app" width={220} height={220}
              style={{ border: '1px solid var(--border)', borderRadius: 8, background: '#fff', padding: 8 }} unoptimized />
          ) : null}
          <div>
            <p className="muted">Manual entry secret</p>
            <div className="copy-row"><code className="code-block">{setup.secret}</code><CopyButton value={setup.secret} label="Copy secret" /></div>
          </div>
          <form action={enableAction} className="stack" noValidate>
            {enableState.error ? <Alert variant="danger">{enableState.error}</Alert> : null}
            <div className="field" style={{ maxWidth: 240 }}>
              <label htmlFor="mfa-code">2. Enter the 6-digit code</label>
              <input id="mfa-code" name="code" inputMode="numeric" pattern="[0-9]*" autoComplete="one-time-code"
                required className="input" placeholder="123456" />
            </div>
            <div><SubmitButton pendingLabel="Verifying…">Enable two-factor</SubmitButton></div>
          </form>
        </div>
      )}
    </div>
  );
}
