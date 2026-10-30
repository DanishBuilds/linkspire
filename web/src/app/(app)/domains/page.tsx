import type { Metadata } from 'next';
import * as api from '@/lib/api';
import { ROLE_RANK } from '@/lib/types';
import { DomainStateBadge, Badge } from '@/components/Badge';
import { CopyButton } from '@/components/CopyButton';
import { Alert } from '@/components/Alert';
import { AddDomainForm } from '@/components/resources/AddDomainForm';
import { Icon } from '@/components/icons';
import { fmtDateTime } from '@/lib/format';

export const metadata: Metadata = { title: 'Domains' };

export default async function DomainsPage() {
  const [domains, me] = await Promise.all([api.listDomains(), api.me()]);
  const role = me.workspaces[0]?.role ?? 'viewer';
  const canAdmin = ROLE_RANK[role] >= ROLE_RANK.admin;

  return (
    <div className="stack">
      <div className="page-head">
        <h1>Custom domains</h1>
        <p>Serve short links from your own domain. Verification and TLS are automatic once DNS is in place.</p>
      </div>

      {canAdmin ? (
        <section className="card" aria-labelledby="add-h">
          <h2 id="add-h" style={{ marginBottom: 14 }}>Add a domain</h2>
          <AddDomainForm />
        </section>
      ) : (
        <Alert variant="info" title="Admin access required">Adding domains needs an admin role.</Alert>
      )}

      <section className="stack" aria-labelledby="dl-h">
        <h2 id="dl-h">Your domains ({domains.length})</h2>
        {domains.length === 0 ? (
          <div className="card empty"><h3>No custom domains</h3><p>Add one above to brand your links.</p></div>
        ) : (
          domains.map((d) => (
            <section className={`card domain-card state-${d.state}`} key={d.id} aria-labelledby={`dom-${d.id}`}>
              <div className="spread" style={{ flexWrap: 'wrap', gap: 12 }}>
                <div className="domain-head">
                  <span className="globe" aria-hidden="true"><Icon.Globe /></span>
                  <span id={`dom-${d.id}`} className="hn">{d.hostname}</span>
                </div>
                <div className="cluster">
                  {d.isApex ? <Badge>apex</Badge> : null}
                  {d.hsts ? <Badge tone="ok">HSTS</Badge> : null}
                  <DomainStateBadge state={d.state} />
                </div>
              </div>
              <p className="muted" style={{ marginTop: 10 }}>
                {d.state === 'active'
                  ? `Verified ${fmtDateTime(d.verifiedAt)} · serving traffic with TLS.`
                  : d.state === 'failed'
                  ? 'Verification failed. Check the DNS records below and retry.'
                  : `Last checked ${fmtDateTime(d.lastChecked)} · next check ${fmtDateTime(d.nextCheck)}.`}
              </p>

              {d.state !== 'active' ? (
                <details style={{ marginTop: 12 }} open={d.state === 'pending' || d.state === 'failed'}>
                  <summary style={{ cursor: 'pointer', fontWeight: 600, minHeight: 44, display: 'flex', alignItems: 'center' }}>
                    DNS records to add
                  </summary>
                  <div className="dns-grid">
                    <div className="dns-row">
                      <p className="k">TXT record (ownership)</p>
                      <div className="copy-row"><code className="code-block">{`_linkspire-verify.${d.hostname}  TXT  ${d.verifyToken}`}</code><CopyButton value={d.verifyToken} label="Copy value" /></div>
                    </div>
                    <div className="dns-row">
                      <p className="k">{d.isApex ? 'ALIAS/ANAME record (apex cannot CNAME)' : 'CNAME record (routing)'}</p>
                      <div className="copy-row"><code className="code-block">{`${d.hostname}  ${d.isApex ? 'ALIAS' : 'CNAME'}  ${d.cnameTarget}`}</code><CopyButton value={d.cnameTarget} label="Copy target" /></div>
                    </div>
                    {d.isApex ? <Alert variant="info">Apex domains cannot use CNAME (RFC 1034). Use your provider’s ALIAS/ANAME, or A-records to the anycast IPs.</Alert> : null}
                  </div>
                </details>
              ) : null}
            </section>
          ))
        )}
      </section>
    </div>
  );
}
