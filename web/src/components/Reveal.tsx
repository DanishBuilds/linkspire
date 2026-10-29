'use client';
import { useState } from 'react';

// Toggles visibility of a sensitive one-time value (e.g. a new API key).
export function Reveal({ value }: { value: string }) {
  const [shown, setShown] = useState(false);
  return (
    <div className="copy-row">
      <code className="code-block" aria-label={shown ? 'secret value' : 'secret value, hidden'}>
        {shown ? value : '•'.repeat(Math.min(value.length, 44))}
      </code>
      <button type="button" className="btn secondary small" onClick={() => setShown((s) => !s)} aria-pressed={shown}>
        {shown ? 'Hide' : 'Show'}
      </button>
    </div>
  );
}
