import Link from 'next/link';
import type { Metadata } from 'next';
import * as api from '@/lib/api';
import type { AnalyticsRow, Link as LinkModel } from '@/lib/types';
import { byDay, byDimension, totals } from '@/lib/analytics';
import { StatCard } from '@/components/StatCard';
import { ShortLinkCell } from '@/components/ShortLinkCell';
import { TimeSeries } from '@/components/charts/TimeSeries';
import { BarList } from '@/components/charts/BarList';
import { Alert } from '@/components/Alert';
import { Icon } from '@/components/icons';
import { fmtNumber } from '@/lib/format';

export const metadata: Metadata = { title: 'Analytics' };

const SAMPLE = 25; // links to roll up (keeps the fan-out bounded)

export default async function InsightsPage() {
  const links = await api.listLinks(100);

  if (links.length === 0) {
    return (
      <div className="stack">
        <div className="page-head"><h1>Analytics</h1><p>Privacy-safe aggregates across your links.</p></div>
        <div className="card empty">
          <h3>No links yet</h3>
          <p>Create a link, then open it through the redirect to start collecting analytics.</p>
          <p style={{ marginTop: 14 }}><Link href="/links" className="btn"><Icon.Plus />Create link</Link></p>
        </div>
      </div>
    );
  }

  // Probe one link first so an unconfigured store short-circuits the fan-out.
  const first = links[0]!;
  const probe = await api.analytics(first.domainId, first.code);

  const header = (
    <div className="page-head">
      <h1>Analytics</h1>
      <p>Privacy-safe aggregates across your {Math.min(links.length, SAMPLE)} most recent links. No IP or user-agent is ever stored.</p>
    </div>
  );

  if (probe.status === 'unconfigured') {
    return (
      <div className="stack">
        {header}
        <Alert variant="info" title="Analytics store is not configured">
          Aggregates need the analytics store running. Start ClickHouse and Redpanda and set
          {' '}<span className="mono">CLICKHOUSE_URL</span> on the API (the docker compose stack wires this up).
        </Alert>
      </div>
    );
  }

  // Roll up the sample. The probe result is reused for the first link.
  const sample = links.slice(0, SAMPLE);
  const rest = await Promise.all(
    sample.slice(1).map((l) =>
      api.analytics(l.domainId, l.code).then((a) => a.rows).catch(() => [] as AnalyticsRow[])),
  );
  const perLink: { link: LinkModel; rows: AnalyticsRow[] }[] = [
    { link: first, rows: probe.rows },
    ...sample.slice(1).map((l, i) => ({ link: l, rows: rest[i] ?? [] })),
  ];

  const allRows = perLink.flatMap((p) => p.rows);
  const daily = byDay(allRows);
  const t = totals(allRows);
  const withTraffic = perLink.filter((p) => p.rows.length > 0).length;
  const topCountry = byDimension(allRows, 'countryCode', 1)[0]?.key ?? '—';
  const topLinks = perLink
    .map((p) => ({ link: p.link, clicks: p.rows.reduce((s, r) => s + r.clicks, 0) }))
    .filter((x) => x.clicks > 0)
    .sort((a, b) => b.clicks - a.clicks)
    .slice(0, 8);

  return (
    <div className="stack">
      {header}

      <div className="grid cols-4">
        <StatCard icon={<Icon.Chart />} value={fmtNumber(t.clicks)} label="Total clicks" />
        <StatCard icon={<Icon.Shield />} value={fmtNumber(t.uniques)} label="Unique visitors (per-day sum)" tone="ok" />
        <StatCard icon={<Icon.Link />} value={fmtNumber(withTraffic)} label="Links with traffic" />
        <StatCard icon={<Icon.Globe />} value={topCountry} label="Top country" />
      </div>

      {daily.length === 0 ? (
        <div className="card empty">
          <h3>No clicks recorded yet</h3>
          <p>Analytics appear after a short link is opened through the redirect service.</p>
          <p className="muted" style={{ marginTop: 10 }}>
            Open one of your short links in a browser, then refresh. Creating, copying, or viewing a link is not a
            click, and events are processed asynchronously (redirect → stream → analytics store).
          </p>
        </div>
      ) : (
        <>
          <section className="card" aria-labelledby="ts-h">
            <h2 id="ts-h" style={{ marginBottom: 10 }}>Clicks over time</h2>
            <TimeSeries points={daily} />
          </section>

          <section className="stack" aria-labelledby="top-h">
            <h2 id="top-h">Top links</h2>
            <div className="table-wrap">
              <table className="data">
                <caption className="visually-hidden">Links ranked by clicks</caption>
                <thead>
                  <tr><th scope="col">Short link</th><th scope="col">Destination</th><th scope="col">Clicks</th></tr>
                </thead>
                <tbody>
                  {topLinks.map(({ link, clicks }) => (
                    <tr key={`${link.domainId}:${link.code}`}>
                      <td><ShortLinkCell domainId={link.domainId} code={link.code} shortUrl={link.shortUrl} /></td>
                      <td className="cell-dest"><a className="destlink" href={link.destination} target="_blank" rel="noopener noreferrer" title={link.destination}>{link.destination}</a></td>
                      <td className="nowrap" style={{ fontVariantNumeric: 'tabular-nums', fontWeight: 600 }}>{fmtNumber(clicks)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>

          <section className="stack" aria-labelledby="bd-h">
            <h2 id="bd-h">Breakdowns</h2>
            <div className="grid cols-3">
              <BarList title="Countries" data={byDimension(allRows, 'countryCode')} />
              <BarList title="Devices" data={byDimension(allRows, 'deviceClass')} />
              <BarList title="Browsers" data={byDimension(allRows, 'browserFamily')} />
              <BarList title="Operating systems" data={byDimension(allRows, 'osFamily')} />
              <BarList title="Referrers" data={byDimension(allRows, 'referrerDomain')} />
              <BarList title="Source" data={byDimension(allRows, 'source')} />
            </div>
          </section>
        </>
      )}
    </div>
  );
}
