// src/components/Badge.tsx — status badges. Color is paired with text and a
// shape so meaning never relies on color alone (WCAG 1.4.1).
import type { ReactNode } from 'react';

type Tone = 'neutral' | 'ok' | 'warn' | 'bad' | 'info';
const toneClass: Record<Tone, string> = { neutral: '', ok: 'ok', warn: 'warn', bad: 'bad', info: 'info' };

export function Badge({ tone = 'neutral', children }: { tone?: Tone; children: ReactNode }) {
  return <span className={`badge ${toneClass[tone]}`}><span className="dot" aria-hidden="true" />{children}</span>;
}

export function LinkStatusBadge({ status }: { status: string }) {
  const map: Record<string, Tone> = {
    active: 'ok', password_protected: 'info', disabled: 'warn', expired: 'bad',
  };
  const label = status.replace(/_/g, ' ');
  return <Badge tone={map[status] ?? 'neutral'}>{label}</Badge>;
}

export function VerdictBadge({ verdict }: { verdict: string }) {
  const map: Record<string, Tone> = {
    clean: 'ok', pending: 'neutral', suspicious: 'warn', malicious: 'bad',
  };
  return <Badge tone={map[verdict] ?? 'neutral'}>{verdict}</Badge>;
}

export function DomainStateBadge({ state }: { state: string }) {
  const map: Record<string, Tone> = {
    active: 'ok', verified: 'info', pending: 'warn', failed: 'bad',
  };
  return <Badge tone={map[state] ?? 'neutral'}>{state}</Badge>;
}

export function DsarStatusBadge({ status }: { status: string }) {
  const map: Record<string, Tone> = {
    completed: 'ok', received: 'info', in_progress: 'warn', rejected: 'bad',
  };
  return <Badge tone={map[status] ?? 'neutral'}>{status.replace(/_/g, ' ')}</Badge>;
}
