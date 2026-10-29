import Link from 'next/link';
import type { Metadata } from 'next';
import * as api from '@/lib/api';
import { ROLE_RANK } from '@/lib/types';
import { CreateLinkForm } from '@/components/links/CreateLinkForm';
import { LinkStatusBadge, VerdictBadge, Badge } from '@/components/Badge';
import { CopyButton } from '@/components/CopyButton';
import { ConfirmSubmit } from '@/components/ConfirmSubmit';
import { deleteLinkAction } from '@/app/actions/links';
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
        <p>Create and manage short links. Destinations are scanned before a link is saved.</p>
      </div>

      <section className="card" aria-labelledby="create-h">
        <h2 id="create-h" style={{ marginBottom: 14 }}>Create a link</h2>
        <CreateLinkForm domains={domains} canWrite={canWrite} />
      </section>

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
                  <th scope="col">Expires</th>
                  <th scope="col">Created</th>
                  {canWrite ? <th scope="col"><span className="visually-hidden">Actions</span></th> : null}
                </tr>
              </thead>
              <tbody>
                {links.map((l) => (
                  <tr key={`${l.domainId}:${l.code}`}>
                    <td>
                      <div className="cluster">
                        <Link href={`/links/${l.domainId}/${encodeURIComponent(l.code)}`} className="mono">{l.code}</Link>
                        <CopyButton value={l.shortUrl} label="Copy" />
                      </div>
                    </td>
                    <td><span className="truncate" title={l.destination}>{l.destination}</span></td>
                    <td>
                      <LinkStatusBadge status={l.status} />
                      {isExpired(l.expiresAt) ? <> <Badge tone="bad">expired</Badge></> : null}
                    </td>
                    <td><VerdictBadge verdict={l.scanVerdict} /></td>
                    <td className="nowrap">{l.redirectType}</td>
                    <td className="nowrap muted">{fmtDate(l.expiresAt)}</td>
                    <td className="nowrap muted">{fmtDate(l.createdAt)}</td>
                    {canWrite ? (
                      <td>
                        <form action={deleteLinkAction}>
                          <input type="hidden" name="domain_id" value={l.domainId} />
                          <input type="hidden" name="code" value={l.code} />
                          <ConfirmSubmit confirm={`Delete link ${l.code}? This cannot be undone.`}>Delete</ConfirmSubmit>
                        </form>
                      </td>
                    ) : null}
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
