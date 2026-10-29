'use client';
import { useFormStatus } from 'react-dom';
import type { ReactNode } from 'react';

// Submit button that reflects pending form state (disabled + busy label).
export function SubmitButton({
  children, pendingLabel = 'Working…', className = 'btn',
}: { children: ReactNode; pendingLabel?: string; className?: string }) {
  const { pending } = useFormStatus();
  return (
    <button type="submit" className={className} disabled={pending} aria-busy={pending}>
      {pending ? pendingLabel : children}
    </button>
  );
}
