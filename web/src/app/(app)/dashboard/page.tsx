import Link from 'next/link';
import type { Metadata } from 'next';
import * as api from '@/lib/api';
import { Icon } from '@/components/icons';
import { LinkStatusBadge, VerdictBadge } from '@/components/Badge';
import { CopyButton } from '@/components/CopyButton';
import { fmtDate, fmtNumber } from '@/lib/format';

export const metadata: Metadata = { title: 'Overview' };

export default async function DashboardPage() {
  const [links, domains] = await Promise.all([api.listLinks(100), api.listDomains()]);
  const active = links.filter((l) => l.status === 'active').length;
  const flagged = links.filter((l) => l.scanVerdict === 'suspicious' || l.scanVerdict === 'malicious').length;
  const activeDomains = domains.filter((d) => d.state === 'active').length;
  const recent = links.slice(0, 8);

  return (
    <div className="stack">
      <div className="page-head spread">
        <div>
          <h1>Overview</h1>
          <p>A snapshot of your workspace.</p>
        </div>
        <Link href="/links" className="btn"><Icon.Plus />Create link</Link>
      </div>

      <div className="grid cols-4">
        <div className="card stat"><span className="value">{fmtNumber(links.length)}</span><span className="label">Total links</span></div>
        <div className="card stat"><span className="value">{fmtNumber(active)}</span><span className="label">Active</span></div>
        <div className="card stat"><span className="value">{fmtNumber(flagged)}</span><span className="label">Flagged by scanner</span></div>
        <div className="card stat"><span className="value">{fmtNumber(activeDomains)}</span><span className="label">Active domains</span></div>
      </div>

      <section className="stack" aria-labelledby="recent-h">
        <div className="spread">
          <h2 id="recent-h">Recent links</h2>
          <Link href="/links">View all</Link>
        </div>
        {recent.length === 0 ? (
          <div className="card empty">
            <h3>No links yet</h3>
            <p>Create your first short link to get started.</p>
            <p style={{ marginTop: 14 }}><Link href="/links" className="btn"><Icon.Plus />Create link</Link></p>
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
                  <th scope="col">Scan</th>
                  <th scope="col">Created</th>
                </tr>
              </thead>
              <tbody>
                {recent.map((l) => (
                  <tr key={`${l.domainId}:${l.code}`}>
                    <td>
                      <div className="cluster">
                        <Link href={`/links/${l.domainId}/${encodeURIComponent(l.code)}`} className="mono">{l.code}</Link>
                        <CopyButton value={l.shortUrl} label="Copy" />
                      </div>
                    </td>
                    <td><span className="truncate" title={l.destination}>{l.destination}</span></td>
                    <td><LinkStatusBadge status={l.status} /></td>
                    <td><VerdictBadge verdict={l.scanVerdict} /></td>
                    <td className="nowrap muted">{fmtDate(l.createdAt)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
