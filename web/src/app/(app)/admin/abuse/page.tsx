import { notFound } from 'next/navigation';
import type { Metadata } from 'next';
import * as api from '@/lib/api';
import { Badge } from '@/components/Badge';
import { SubmitButton } from '@/components/SubmitButton';
import { resolveAbuseAction } from '@/app/actions/admin';
import { fmtDateTime } from '@/lib/format';

export const metadata: Metadata = { title: 'Abuse review' };

const STATUS_TONE: Record<string, 'warn' | 'info' | 'ok' | 'bad'> = {
  open: 'warn', reviewing: 'info', actioned: 'ok', dismissed: 'bad',
};

export default async function AbuseQueuePage({ searchParams }: { searchParams: Promise<{ status?: string }> }) {
  const me = await api.me();
  if (!me.isStaff) notFound();
  const sp = await searchParams;
  const status = sp.status ?? 'open';
  const reports = await api.listAbuse(status === 'all' ? '' : status);

  const filters = ['open', 'reviewing', 'actioned', 'dismissed', 'all'];

  return (
    <div className="stack">
      <div className="page-head">
        <h1>Abuse review</h1>
        <p>Reports submitted through the public abuse form. Staff only.</p>
      </div>

      <nav className="tabs" aria-label="Filter by status">
        {filters.map((f) => (
          <a key={f} href={`/admin/abuse?status=${f}`} aria-current={f === status ? 'page' : undefined}>{f}</a>
        ))}
      </nav>

      {reports.length === 0 ? (
        <div className="card empty"><h3>Nothing here</h3><p>No reports with this status.</p></div>
      ) : (
        <div className="table-wrap">
          <table className="data">
            <caption className="visually-hidden">Abuse reports</caption>
            <thead>
              <tr><th scope="col">Short link</th><th scope="col">Reason</th><th scope="col">Status</th><th scope="col">Filed</th><th scope="col">Action</th></tr>
            </thead>
            <tbody>
              {reports.map((rep) => (
                <tr key={rep.id}>
                  <td><span className="truncate mono" title={rep.shortUrl}>{rep.shortUrl}</span></td>
                  <td><span className="truncate" title={rep.reason}>{rep.reason}</span></td>
                  <td><Badge tone={STATUS_TONE[rep.status] ?? 'neutral'}>{rep.status}</Badge></td>
                  <td className="nowrap muted">{fmtDateTime(rep.createdAt)}</td>
                  <td>
                    <form action={resolveAbuseAction} className="cluster" style={{ gap: 6 }}>
                      <input type="hidden" name="id" value={rep.id} />
                      <button type="submit" name="status" value="reviewing" className="btn secondary small">Reviewing</button>
                      <button type="submit" name="status" value="actioned" className="btn small">Action</button>
                      <button type="submit" name="status" value="dismissed" className="btn secondary small">Dismiss</button>
                    </form>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
