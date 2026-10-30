import type { ReactNode } from 'react';
import { Sparkline } from '@/components/charts/Sparkline';

type Tone = 'accent' | 'ok' | 'warn' | 'bad';

// A metric tile: tinted icon chip, big value, label, optional delta + sparkline.
export function StatCard({
  icon, value, label, tone = 'accent', delta, spark,
}: { icon: ReactNode; value: string; label: string; tone?: Tone; delta?: string; spark?: number[] }) {
  return (
    <div className="card stat-card">
      <span className={`chip ${tone === 'accent' ? '' : tone}`} aria-hidden="true">{icon}</span>
      <span className="meta">
        <span className="value">{value}</span>
        <span className="label">{label}</span>
      </span>
      {delta ? <span className="stat-delta">{delta}</span> : null}
      {spark && spark.length > 1 ? <span className="stat-spark"><Sparkline data={spark} /></span> : null}
    </div>
  );
}
