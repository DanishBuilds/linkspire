'use client';
import { useState, useRef } from 'react';
import { Icon } from './icons';

// Copy-to-clipboard with an aria-live confirmation. `compact` renders a square
// icon-only button for use inside dense table cells. Falls back to a hidden
// textarea where the async clipboard API is unavailable.
export function CopyButton({
  value, label = 'Copy', compact = false,
}: { value: string; label?: string; compact?: boolean }) {
  const [copied, setCopied] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  async function copy() {
    let ok = false;
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(value);
        ok = true;
      }
    } catch { ok = false; }
    if (!ok) {
      try {
        const ta = document.createElement('textarea');
        ta.value = value; ta.style.position = 'fixed'; ta.style.opacity = '0';
        document.body.appendChild(ta); ta.select();
        ok = document.execCommand('copy');
        document.body.removeChild(ta);
      } catch { ok = false; }
    }
    setCopied(ok);
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(() => setCopied(false), 2000);
  }

  if (compact) {
    return (
      <button type="button" className="icon-btn" onClick={copy} aria-label={copied ? 'Copied' : label} title={label}>
        {copied ? <Icon.Check /> : <Icon.Copy />}
        <span aria-live="polite" className="visually-hidden">{copied ? `${label} succeeded` : ''}</span>
      </button>
    );
  }

  return (
    <button type="button" className="btn secondary small" onClick={copy}>
      {copied ? <Icon.Check /> : <Icon.Copy />}
      <span>{copied ? 'Copied' : label}</span>
      <span aria-live="polite" className="visually-hidden">{copied ? `${label} succeeded` : ''}</span>
    </button>
  );
}
