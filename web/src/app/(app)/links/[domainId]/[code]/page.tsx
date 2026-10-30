import Link from 'next/link';
import { notFound } from 'next/navigation';
import type { Metadata } from 'next';
import * as api from '@/lib/api';
import { ApiError } from '@/lib/api';
import { ROLE_RANK } from '@/lib/types';
import { LinkStatusBadge, VerdictBadge, Badge } from '@/components/Badge';
import { CopyButton } from '@/components/CopyButton';
import { ConfirmSubmit } from '@/components/ConfirmSubmit';
import { Icon } from '@/components/icons';
import { Alert } from '@/components/Alert';
import { EditLinkForm } from '@/components/links/EditLinkForm';
import { LinkSubnav } from '@/components/links/LinkSubnav';
import { QrPanel } from '@/components/links/QrPanel';
import { TimeSeries } from '@/components/charts/TimeSeries';
import { BarList } from '@/components/charts/BarList';
import { deleteLinkAction } from '@/app/actions/links';
import { byDay, byDimension, totals } from '@/lib/analytics';
import { fmtDateTime, fmtNumber } from '@/lib/format';

export const metadata: Metadata = { title: 'Link detail' };

export default async function LinkDetailPage({ params }: { params: Promise<{ domainId: string; code: string }> }) {
  const { domainId: did, code: rawCode } = await params;
  const domainId = Number(did);
  const code = decodeURIComponent(rawCode);

  let link;
  try {
    link = await api.getLink(domainId, code);
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) notFound();
    throw e;
  }
  const [analytics, me] = await Promise.all([
    api.analytics(domainId, code),
    api.me(),
  ]);
  const role = me.workspaces[0]?.role ?? 'viewer';
  const canWrite = ROLE_RANK[role] >= ROLE_RANK.member;

  const daily = byDay(analytics.rows);
  const t = totals(analytics.rows);

  return (
    <div className="stack">
      <div>
        <Link href="/links" className="muted">← Back to links</Link>
      </div>
      <div className="page-head spread">
        <div>
          <h1 className="mono">{link.code}</h1>
          <div className="cluster" style={{ marginTop: 8 }}>
            <LinkStatusBadge status={link.status} />
            <VerdictBadge verdict={link.scanVerdict} />
            {link.passwordProtected ? <Badge tone="info">password</Badge> : null}
          </div>
        </div>
        <div className="cluster">
          <CopyButton value={link.shortUrl} label="Copy short link" />
          <a className="btn secondary" href={`#qr-code`}><Icon.Qr /> QR code</a>
          <a className="btn secondary" href={link.shortUrl} target="_blank" rel="noopener noreferrer"><Icon.External />Open</a>
        </div>
      </div>

      <LinkSubnav />

      <section className="card" id="link-details" aria-labelledby="det-h">
        <h2 id="det-h">Details</h2>
        <dl className="grid cols-2" style={{ marginTop: 12 }}>
          <div><dt className="muted">Short link</dt><dd className="mono">{link.shortUrl}</dd></div>
          <div><dt className="muted">Destination</dt><dd style={{ wordBreak: 'break-all' }}>{link.destination}</dd></div>
          <div><dt className="muted">Redirect type</dt><dd>{link.redirectType}</dd></div>
          <div><dt className="muted">Created</dt><dd>{fmtDateTime(link.createdAt)}</dd></div>
          <div><dt className="muted">Expires</dt><dd>{fmtDateTime(link.expiresAt)}</dd></div>
          <div><dt className="muted">Click limit</dt><dd>{link.clickLimit != null ? fmtNumber(link.clickLimit) : 'None'}</dd></div>
        </dl>
      </section>

      <QrPanel link={link} />

      <section className="stack" id="link-analytics" aria-labelledby="an-h">
        <h2 id="an-h">Analytics</h2>
        {analytics.status === 'unconfigured' ? (
          <Alert variant="info" title="Analytics store is not configured">
            Aggregates need the analytics store running. Start ClickHouse and Redpanda and set
            {' '}<span className="mono">CLICKHOUSE_URL</span> on the API service (the docker compose stack wires
            this up automatically). No raw IP or user-agent is ever stored; unique visitors are per-day only.
          </Alert>
        ) : analytics.status === 'error' ? (
          <Alert variant="warning" title="Analytics backend is not reachable right now">
            The API could not read ClickHouse. Check that ClickHouse is running and reachable from the API,
            then refresh. {analytics.note}
          </Alert>
        ) : daily.length === 0 ? (
          <div className="card empty">
            <h3>No clicks recorded yet</h3>
            <p>Analytics appear only after someone opens the short link through the redirect service.</p>
            <p className="muted" style={{ marginTop: 10 }}>
              To generate a data point, open <span className="mono">{link.shortUrl}</span> in a browser, then refresh
              this page after a few seconds. Creating, copying, or viewing a link does not count as a click, and
              events are processed asynchronously (redirect → stream → analytics store).
            </p>
          </div>
        ) : (
          <>
            <div className="grid cols-3">
              <div className="card stat"><span className="value">{fmtNumber(t.clicks)}</span><span className="label">Total clicks</span></div>
              <div className="card stat"><span className="value">{fmtNumber(t.uniques)}</span><span className="label">Unique visitors (per-day sum)</span></div>
              <div className="card stat"><span className="value">{fmtNumber(t.days)}</span><span className="label">Days with traffic</span></div>
            </div>
            <div className="card">
              <h3 style={{ marginBottom: 10 }}>Clicks over time</h3>
              <TimeSeries points={daily} />
            </div>
            <div className="grid cols-3">
              <BarList title="Countries" data={byDimension(analytics.rows, 'countryCode')} />
              <BarList title="Devices" data={byDimension(analytics.rows, 'deviceClass')} />
              <BarList title="Browsers" data={byDimension(analytics.rows, 'browserFamily')} />
              <BarList title="Operating systems" data={byDimension(analytics.rows, 'osFamily')} />
              <BarList title="Referrers" data={byDimension(analytics.rows, 'referrerDomain')} />
              <BarList title="Source" data={byDimension(analytics.rows, 'source')} />
            </div>
          </>
        )}
      </section>

      {canWrite ? (
        <section className="card" aria-labelledby="edit-h">
          <h2 id="edit-h" style={{ marginBottom: 12 }}>Edit link</h2>
          <EditLinkForm link={link} />
        </section>
      ) : null}

      {canWrite ? (
        <section className="card" aria-labelledby="danger-h">
          <h2 id="danger-h">Danger zone</h2>
          <p className="muted" style={{ margin: '8px 0 14px' }}>Deleting a link stops its redirect. The code is not reused.</p>
          <form action={deleteLinkAction}>
            <input type="hidden" name="domain_id" value={link.domainId} />
            <input type="hidden" name="code" value={link.code} />
            <ConfirmSubmit confirm={`Delete link ${link.code}? This cannot be undone.`} className="btn danger">
              <Icon.Trash />Delete link
            </ConfirmSubmit>
          </form>
        </section>
      ) : null}
    </div>
  );
}
