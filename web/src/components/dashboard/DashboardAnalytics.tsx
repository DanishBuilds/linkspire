'use client';

import { useMemo, useState } from 'react';
import {
  Area, AreaChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis,
} from 'recharts';
import { useReducedMotion } from 'framer-motion';
import type { DailyPoint } from '@/lib/analytics';

const PERIODS = [
  { key: '24h', days: 1, label: '24h' },
  { key: '7d', days: 7, label: '7d' },
  { key: '30d', days: 30, label: '30d' },
  { key: '90d', days: 90, label: '90d' },
  { key: 'all', days: Infinity, label: 'All' },
] as const;

export function DashboardAnalytics({ daily }: { daily: DailyPoint[] }) {
  const reduce = useReducedMotion();
  const [period, setPeriod] = useState<(typeof PERIODS)[number]['key']>('30d');

  const data = useMemo(() => {
    const p = PERIODS.find((x) => x.key === period)!;
    const rows = p.days === Infinity ? daily : daily.slice(-p.days);
    return rows.map((r) => ({ ...r, dateLabel: r.date.slice(5) }));
  }, [daily, period]);

  return (
    <section className="card" aria-labelledby="dash-chart-h">
      <div className="section-bar" style={{ marginBottom: 12 }}>
        <h2 id="dash-chart-h" style={{ fontSize: '1.05rem' }}>Clicks over time</h2>
        <div className="period-switch" role="group" aria-label="Time range">
          {PERIODS.map((p) => (
            <button
              key={p.key}
              type="button"
              className={`period-btn ${period === p.key ? 'active' : ''}`}
              aria-pressed={period === p.key}
              onClick={() => setPeriod(p.key)}
            >
              {p.label}
            </button>
          ))}
        </div>
      </div>

      {data.length === 0 ? (
        <p className="muted" style={{ padding: '24px 0' }}>No clicks recorded in this range yet.</p>
      ) : (
        <div style={{ width: '100%', height: 260 }}>
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={data} margin={{ top: 6, right: 8, bottom: 0, left: 4 }}>
              <defs>
                <linearGradient id="dashClk" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="var(--c1)" stopOpacity={0.35} />
                  <stop offset="100%" stopColor="var(--c1)" stopOpacity={0} />
                </linearGradient>
                <linearGradient id="dashUnq" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="var(--c2)" stopOpacity={0.22} />
                  <stop offset="100%" stopColor="var(--c2)" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid stroke="var(--border)" vertical={false} />
              <XAxis dataKey="dateLabel" tick={{ fill: 'var(--text-muted)', fontSize: 11 }} tickLine={false} axisLine={{ stroke: 'var(--border)' }} minTickGap={24} />
              <YAxis tick={{ fill: 'var(--text-muted)', fontSize: 11 }} tickLine={false} axisLine={false} width={56} />
              <Tooltip
                cursor={{ stroke: 'var(--border-strong)' }}
                contentStyle={{ background: 'var(--surface)', border: '1px solid var(--border-strong)', borderRadius: 10, fontSize: 12, color: 'var(--text)' }}
                labelStyle={{ color: 'var(--text-muted)' }}
              />
              <Area type="monotone" dataKey="uniques" name="Unique / day" stroke="var(--c2)" strokeWidth={2} fill="url(#dashUnq)" isAnimationActive={!reduce} />
              <Area type="monotone" dataKey="clicks" name="Clicks" stroke="var(--c1)" strokeWidth={2} fill="url(#dashClk)" isAnimationActive={!reduce} />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      )}
      <div className="legend" aria-hidden="true" style={{ marginTop: 8 }}>
        <span className="item"><span className="swatch" style={{ background: 'var(--c1)' }} />Clicks</span>
        <span className="item"><span className="swatch" style={{ background: 'var(--c2)' }} />Unique visitors (per day)</span>
      </div>
    </section>
  );
}
