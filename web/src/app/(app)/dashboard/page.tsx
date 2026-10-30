import Link from 'next/link';
import type { Metadata } from 'next';
import * as api from '@/lib/api';
import type { AnalyticsRow, Link as LinkModel } from '@/lib/types';
import { byDay, byDimension, totals } from '@/lib/analytics';
import { Icon } from '@/components/icons';
import { LinkStatusBadge, VerdictBadge } from '@/components/Badge';
import { StatCard } from '@/components/StatCard';
import { ShortLinkCell } from '@/components/ShortLinkCell';
import { BarList } from '@/components/charts/BarList';
import { DashboardAnalytics } from '@/components/dashboard/DashboardAnalytics';
import { CreateLinkButton } from '@/components/links/CreateLinkButton';
import { fmtDate, fmtNumber } from '@/lib/format';

export const metadata: Metadata = { title: 'Overview' };

const SAMPLE = 25;

export default async function DashboardPage() {
  const [links, domains] = await Promise.all([api.listLinks(100), api.listDomains()]);
  const recent = links.slice(0, 8);

  // Roll up analytics across a bounded sample, probing one link first so an
  // unconfigured store short-circuits the fan-out.
  let daily: ReturnType<typeof byDay> = [];
  let t = { clicks: 0, uniques: 0, days: 0 };
  let topLink: { link: LinkModel; clicks: number } | null = null;
  let countries: ReturnType<typeof byDimension> = [];
  let referrers: ReturnType<typeof byDimension> = [];
  let configured = links.length > 0;

  if (links.length > 0) {
    const first = links[0]!;
    const probe = await api.analytics(first.domainId, first.code);
    if (probe.status === 'unconfigured') {
      configured = false;
    } else {
      const sample = links.slice(0, SAMPLE);
      const rest = await Promise.all(
        sample.slice(1).map((l) => api.analytics(l.domainId, l.code).then((a) => a.rows).catch(() => [] as AnalyticsRow[])),
      );
      const perLink = [
        { link: first, rows: probe.rows },
        ...sample.slice(1).map((l, i) => ({ link: l, rows: rest[i] ?? [] })),
      ];
      const allRows = perLink.flatMap((p) => p.rows);
      daily = byDay(allRows);
      t = totals(allRows);
      countries = byDimension(allRows, 'countryCode', 6);
      referrers = byDimension(allRows, 'referrerDomain', 6);
      const ranked = perLink
        .map((p) => ({ link: p.link, clicks: p.rows.reduce((s, r) => s + r.clicks, 0) }))
        .sort((a, b) => b.clicks - a.clicks);
      topLink = ranked[0] && ranked[0].clicks > 0 ? ranked[0] : null;
    }
  }

  const spark = daily.slice(-14).map((d) => d.clicks);

  return (
    <div className="stack">
      <div className="page-head spread">
        <div>
          <h1>Overview</h1>
          <p>A snapshot of your workspace.</p>
        </div>
        <CreateLinkButton domains={domains} />
      </div>

      <div className="grid cols-4">
        <StatCard icon={<Icon.Link />} value={fmtNumber(links.length)} label="Total links" />
        <StatCard icon={<Icon.Chart />} value={fmtNumber(t.clicks)} label="Total clicks" tone="ok" spark={spark.length > 1 ? spark : undefined} />
        <StatCard icon={<Icon.Shield />} value={fmtNumber(t.uniques)} label="Unique visitors" />
        <StatCard icon={<Icon.Qr />} value={topLink ? topLink.link.code : '—'} label="Top performing link" />
      </div>

      {configured ? (
        <DashboardAnalytics daily={daily} />
      ) : (
        <div className="card empty">
          <h3>Analytics store not configured</h3>
          <p>Start ClickHouse and Redpanda and set <span className="mono">CLICKHOUSE_URL</span> on the API to see clicks over time.</p>
        </div>
      )}

      <div className="grid cols-2-1">
        <section className="stack" aria-labelledby="recent-h">
          <div className="section-bar">
            <h2 id="recent-h">Recent links</h2>
            <Link href="/links" className="link-more">View all →</Link>
          </div>
          {recent.length === 0 ? (
            <div className="card empty">
              <h3>No links yet</h3>
              <p>Create your first short link to get started.</p>
              <p style={{ marginTop: 14 }}><CreateLinkButton domains={domains} /></p>
            </div>
          ) : (
            <div className="table-wrap">
              <table className="data">
                <caption className="visually-hidden">Your most recently created links</caption>
                <thead>
                  <tr>
                    <th scope="col">Short link</th>
                    <th scope="col">Destination</th>
                    <th scope="col">Status</th>
                    <th scope="col">Created</th>
                  </tr>
                </thead>
                <tbody>
                  {recent.map((l) => (
                    <tr key={`${l.domainId}:${l.code}`}>
                      <td><ShortLinkCell domainId={l.domainId} code={l.code} shortUrl={l.shortUrl} /></td>
                      <td className="cell-dest">
                        <a className="destlink" href={l.destination} target="_blank" rel="noopener noreferrer" title={l.destination}>{l.destination}</a>
                      </td>
                      <td><LinkStatusBadge status={l.status} /></td>
                      <td className="nowrap muted">{fmtDate(l.createdAt)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>

        <div className="stack">
          <BarList title="Top countries" data={countries} />
          <BarList title="Top referrers" data={referrers} />
        </div>
      </div>
    </div>
  );
}
