'use client';
import { useId, useState } from 'react';
import type { DailyPoint } from '@/lib/analytics';
import { fmtNumber } from '@/lib/format';

// Clicks vs unique-visitors over time. Two series on ONE axis (same unit),
// 2px lines, soft area under clicks, hover crosshair + tooltip, and an
// equivalent data table (toggle) so the figure is never color/pointer-only.
const W = 760, H = 260, PL = 48, PR = 14, PT = 14, PB = 30;
const IW = W - PL - PR, IH = H - PT - PB;

export function TimeSeries({ points }: { points: DailyPoint[] }) {
  const titleId = useId();
  const [hover, setHover] = useState<number | null>(null);
  const [showTable, setShowTable] = useState(false);

  if (points.length === 0) {
    return <p className="muted">No daily data to plot yet.</p>;
  }

  const maxY = Math.max(1, ...points.map((p) => Math.max(p.clicks, p.uniques)));
  const n = points.length;
  const x = (i: number) => PL + (n === 1 ? IW / 2 : (i / (n - 1)) * IW);
  const y = (v: number) => PT + IH - (v / maxY) * IH;

  const linePath = (sel: (p: DailyPoint) => number) =>
    points.map((p, i) => `${i === 0 ? 'M' : 'L'}${x(i).toFixed(1)},${y(sel(p)).toFixed(1)}`).join(' ');
  const areaPath =
    `${linePath((p) => p.clicks)} L${x(n - 1).toFixed(1)},${(PT + IH).toFixed(1)} L${x(0).toFixed(1)},${(PT + IH).toFixed(1)} Z`;

  const yTicks = 4;
  const gridY = Array.from({ length: yTicks + 1 }, (_, i) => Math.round((maxY / yTicks) * i));
  const xTickEvery = Math.max(1, Math.ceil(n / 6));

  function onMove(e: React.PointerEvent<SVGSVGElement>) {
    const rect = e.currentTarget.getBoundingClientRect();
    const px = ((e.clientX - rect.left) / rect.width) * W;
    const i = Math.round(((px - PL) / IW) * (n - 1));
    setHover(Math.min(n - 1, Math.max(0, i)));
  }

  const hp = hover != null ? points[hover] : null;

  return (
    <figure style={{ margin: 0 }}>
      <figcaption id={titleId} className="visually-hidden">
        Clicks and per-day unique visitors over {n} day{n === 1 ? '' : 's'}.
      </figcaption>
      <div style={{ position: 'relative' }}>
        <svg
          className="chart" viewBox={`0 0 ${W} ${H}`} width="100%" role="img"
          aria-labelledby={titleId} preserveAspectRatio="xMidYMid meet"
          onPointerMove={onMove} onPointerLeave={() => setHover(null)}
          style={{ touchAction: 'none' }}
        >
          <g className="grid">
            {gridY.map((v) => (
              <g key={v}>
                <line x1={PL} x2={W - PR} y1={y(v)} y2={y(v)} />
                <text x={PL - 8} y={y(v) + 4} textAnchor="end">{fmtNumber(v)}</text>
              </g>
            ))}
          </g>
          <g className="axis">
            {points.map((p, i) =>
              i % xTickEvery === 0 || i === n - 1 ? (
                <text key={p.date} x={x(i)} y={H - 10} textAnchor="middle">{p.date.slice(5)}</text>
              ) : null,
            )}
          </g>
          <path className="area" d={areaPath} />
          <path d={linePath((p) => p.clicks)} fill="none" stroke="var(--c1)" strokeWidth={2} />
          <path d={linePath((p) => p.uniques)} fill="none" stroke="var(--c2)" strokeWidth={2} strokeDasharray="5 4" />
          {hp && hover != null ? (
            <g>
              <line x1={x(hover)} x2={x(hover)} y1={PT} y2={PT + IH} stroke="var(--border-strong)" strokeWidth={1} />
              <circle cx={x(hover)} cy={y(hp.clicks)} r={4.5} fill="var(--c1)" stroke="var(--surface)" strokeWidth={2} />
              <circle cx={x(hover)} cy={y(hp.uniques)} r={4.5} fill="var(--c2)" stroke="var(--surface)" strokeWidth={2} />
            </g>
          ) : null}
        </svg>
        {hp ? (
          <div
            role="status"
            style={{
              position: 'absolute', top: 6, right: 6, background: 'var(--surface)',
              border: '1px solid var(--border-strong)', borderRadius: 8, padding: '8px 10px',
              fontSize: '0.85rem', boxShadow: 'var(--shadow)', pointerEvents: 'none',
            }}
          >
            <strong>{hp.date}</strong>
            <div>Clicks: {fmtNumber(hp.clicks)}</div>
            <div>Unique (per day): {fmtNumber(hp.uniques)}</div>
          </div>
        ) : null}
      </div>

      <div className="legend" aria-hidden="true">
        <span className="item"><span className="swatch" style={{ background: 'var(--c1)' }} />Clicks</span>
        <span className="item"><span className="swatch" style={{ background: 'var(--c2)' }} />Unique visitors (per day)</span>
      </div>

      <div style={{ marginTop: 10 }}>
        <button type="button" className="btn ghost small" onClick={() => setShowTable((s) => !s)} aria-expanded={showTable}>
          {showTable ? 'Hide data table' : 'View as table'}
        </button>
      </div>
      {showTable ? (
        <div className="table-wrap" style={{ marginTop: 10 }}>
          <table className="data">
            <caption className="visually-hidden">Clicks and per-day unique visitors by date</caption>
            <thead><tr><th scope="col">Date</th><th scope="col">Clicks</th><th scope="col">Unique (per day)</th></tr></thead>
            <tbody>
              {points.map((p) => (
                <tr key={p.date}><th scope="row">{p.date}</th><td>{fmtNumber(p.clicks)}</td><td>{fmtNumber(p.uniques)}</td></tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
    </figure>
  );
}
