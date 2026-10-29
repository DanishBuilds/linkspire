// src/lib/analytics.ts — pure aggregation over privacy-safe daily rollups.
import type { AnalyticsRow } from './types';

export interface DailyPoint { date: string; clicks: number; uniques: number; }
export interface DimensionSlice { key: string; clicks: number; }

export function byDay(rows: AnalyticsRow[]): DailyPoint[] {
  const map = new Map<string, DailyPoint>();
  for (const r of rows) {
    if (!r.date) continue;
    const p = map.get(r.date) ?? { date: r.date, clicks: 0, uniques: 0 };
    p.clicks += r.clicks;
    p.uniques += r.uniqueVisitors;
    map.set(r.date, p);
  }
  return [...map.values()].sort((a, b) => a.date.localeCompare(b.date));
}

export function byDimension(
  rows: AnalyticsRow[],
  field: keyof AnalyticsRow,
  topN = 8,
): DimensionSlice[] {
  const map = new Map<string, number>();
  for (const r of rows) {
    const raw = r[field];
    const key = raw == null || raw === '' ? 'unknown' : String(raw);
    map.set(key, (map.get(key) ?? 0) + r.clicks);
  }
  const sorted = [...map.entries()].map(([key, clicks]) => ({ key, clicks })).sort((a, b) => b.clicks - a.clicks);
  if (sorted.length <= topN) return sorted;
  const head = sorted.slice(0, topN);
  const rest = sorted.slice(topN).reduce((s, x) => s + x.clicks, 0);
  if (rest > 0) head.push({ key: 'Other', clicks: rest });
  return head;
}

export function totals(rows: AnalyticsRow[]): { clicks: number; uniques: number; days: number } {
  const days = byDay(rows);
  return {
    clicks: rows.reduce((s, r) => s + r.clicks, 0),
    uniques: days.reduce((s, d) => s + d.uniques, 0), // per-day uniques summed (not cross-day unique by design)
    days: days.length,
  };
}
