// src/components/charts/BarList.tsx — horizontal magnitude bars for a single
// categorical breakdown. One hue (magnitude, not identity); every row carries a
// text label and value, so it never depends on color. Server component.
import type { DimensionSlice } from '@/lib/analytics';
import { fmtNumber } from '@/lib/format';

function slug(s: string) { return s.toLowerCase().replace(/[^a-z0-9]+/g, '-'); }

export function BarList({ title, data }: { title: string; data: DimensionSlice[] }) {
  const max = Math.max(1, ...data.map((d) => d.clicks));
  const total = data.reduce((s, d) => s + d.clicks, 0);
  return (
    <section className="card" aria-labelledby={`bl-${slug(title)}`}>
      <h3 id={`bl-${slug(title)}`}>{title}</h3>
      {data.length === 0 ? (
        <p className="muted" style={{ marginTop: 8 }}>No data.</p>
      ) : (
        <ul className="bar-list" style={{ listStyle: 'none', padding: 0, marginTop: 12 }}>
          {data.map((d) => {
            const pct = total ? Math.round((d.clicks / total) * 100) : 0;
            return (
              <li key={d.key} className="row">
                <span className="k">{d.key}</span>
                <span className="v">{fmtNumber(d.clicks)} ({pct}%)</span>
                <span className="track" aria-hidden="true">
                  <span className="fill" style={{ width: `${(d.clicks / max) * 100}%` }} />
                </span>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}
