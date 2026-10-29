'use client';
import { useFormStatus } from 'react-dom';
import type { ReactNode } from 'react';

// Submit button for destructive actions: requires an explicit confirmation
// before the form posts. Keyboard and pointer both trigger the native confirm.
export function ConfirmSubmit({
  children, confirm, className = 'btn danger small', pendingLabel = 'Working…',
}: { children: ReactNode; confirm: string; className?: string; pendingLabel?: string }) {
  const { pending } = useFormStatus();
  return (
    <button
      type="submit" className={className} disabled={pending} aria-busy={pending}
      onClick={(e) => { if (!window.confirm(confirm)) e.preventDefault(); }}
    >
      {pending ? pendingLabel : children}
    </button>
  );
}
