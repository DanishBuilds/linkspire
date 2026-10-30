import type { ReactNode } from 'react';

type Tone = 'accent' | 'ok' | 'warn' | 'bad';

// A metric tile: tinted icon chip, big value, label. Used on the overview.
export function StatCard({
  icon, value, label, tone = 'accent',
}: { icon: ReactNode; value: string; label: string; tone?: Tone }) {
  return (
    <div className="card stat-card">
      <span className={`chip ${tone === 'accent' ? '' : tone}`} aria-hidden="true">{icon}</span>
      <span className="meta">
        <span className="value">{value}</span>
        <span className="label">{label}</span>
      </span>
    </div>
  );
}
