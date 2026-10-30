import Link from 'next/link';
import type { Metadata } from 'next';
import * as api from '@/lib/api';
import { ROLE_RANK } from '@/lib/types';
import { CreateLinkForm } from '@/components/links/CreateLinkForm';
import { BulkCreateForm } from '@/components/links/BulkCreateForm';
import { LinkStatusBadge, VerdictBadge, Badge } from '@/components/Badge';
import { ConfirmSubmit } from '@/components/ConfirmSubmit';
import { ShortLinkCell } from '@/components/ShortLinkCell';
import { deleteLinkAction } from '@/app/actions/links';
import { Icon } from '@/components/icons';
import { fmtDate, isExpired } from '@/lib/format';

export const metadata: Metadata = { title: 'Links' };

export default async function LinksPage() {
  const [links, domains, me] = await Promise.all([api.listLinks(100), api.listDomains(), api.me()]);
  const role = me.workspaces[0]?.role ?? 'viewer';
  const canWrite = ROLE_RANK[role] >= ROLE_RANK.member;

  return (
    <div className="stack">
      <div className="page-head">
        <h1>Links</h1>
        <ol className="links-howto muted">
          <li><strong>Create a link</strong> below. Optional password: expand “Password protection, expiry &amp; click limits”.</li>
          <li><strong>Get a QR code:</strong> in the table, click the purple <strong>QR code</strong> button (or “Create QR code” right after you create a link).</li>
        </ol>
      </div>

      <section className="card" aria-labelledby="create-h">
        <h2 id="create-h" style={{ marginBottom: 14 }}>Create a link</h2>
        <CreateLinkForm domains={domains} canWrite={canWrite} />
      </section>

      {canWrite ? (
        <section className="card" aria-labelledby="bulk-h">
          <details>
            <summary style={{ cursor: 'pointer', fontWeight: 650, minHeight: 44, display: 'flex', alignItems: 'center' }}>
              <h2 id="bulk-h" style={{ fontSize: '1.25rem', margin: 0 }}>Bulk create</h2>
            </summary>
            <div style={{ marginTop: 14 }}>
              <BulkCreateForm />
            </div>
          </details>
        </section>
      ) : null}

      <section className="stack" aria-labelledby="all-h">
        <h2 id="all-h">All links ({links.length})</h2>
        {links.length === 0 ? (
          <div className="card empty"><h3>No links yet</h3><p>Links you create appear here.</p></div>
        ) : (
          <div className="table-wrap">
            <table className="data">
              <caption className="visually-hidden">All links in this workspace</caption>
              <thead>
                <tr>
                  <th scope="col">Short link</th>
                  <th scope="col">Destination</th>
                  <th scope="col">Status</th>
                  <th scope="col">Scan</th>
                  <th scope="col">Redirect</th>
                  <th scope="col">Created</th>
                  <th scope="col" className="col-actions"><span className="visually-hidden">Actions</span></th>
                </tr>
              </thead>
              <tbody>
                {links.map((l) => (
                  <tr key={`${l.domainId}:${l.code}`}>
                    <td><ShortLinkCell domainId={l.domainId} code={l.code} shortUrl={l.shortUrl} /></td>
                    <td className="cell-dest">
                      <a className="destlink" href={l.destination} target="_blank" rel="noopener noreferrer" title={l.destination}>{l.destination}</a>
                    </td>
                    <td>
                      <span className="badge-stack">
                        <LinkStatusBadge status={l.status} />
                        {l.passwordProtected ? <Badge tone="info">password</Badge> : null}
                        {isExpired(l.expiresAt) ? <Badge tone="bad">expired</Badge> : null}
                      </span>
                    </td>
                    <td><VerdictBadge verdict={l.scanVerdict} /></td>
                    <td className="nowrap">{l.redirectType}</td>
                    <td className="nowrap muted">{fmtDate(l.createdAt)}</td>
                    <td className="col-actions">
                      <span className="row-actions">
                        <Link className="icon-btn" href={`/links/${l.domainId}/${encodeURIComponent(l.code)}#qr-code`} title="QR code" aria-label={`QR code for ${l.code}`}>
                          <Icon.Qr />
                        </Link>
                        {canWrite ? (
                          <form action={deleteLinkAction}>
                            <input type="hidden" name="domain_id" value={l.domainId} />
                            <input type="hidden" name="code" value={l.code} />
                            <ConfirmSubmit confirm={`Delete link ${l.code}? This cannot be undone.`} className="icon-btn" pendingLabel="…">
                              <Icon.Trash /><span className="visually-hidden">Delete {l.code}</span>
                            </ConfirmSubmit>
                          </form>
                        ) : null}
                      </span>
                    </td>
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
