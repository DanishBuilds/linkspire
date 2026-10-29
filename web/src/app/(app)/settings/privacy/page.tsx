import type { Metadata } from 'next';
import * as api from '@/lib/api';
import { DsarStatusBadge } from '@/components/Badge';
import { DsarForm } from '@/components/account/DsarForm';
import { DoNotSellForm } from '@/components/account/DoNotSellForm';
import { fmtDate } from '@/lib/format';

export const metadata: Metadata = { title: 'Privacy' };

export default async function PrivacyPage() {
  const [requests, me] = await Promise.all([api.listDsar(), api.me()]);
  return (
    <div className="stack">
      <section className="card" aria-labelledby="dns-h">
        <h2 id="dns-h" style={{ marginBottom: 12 }}>Do Not Sell or Share</h2>
        <DoNotSellForm alreadySet={me.gpcOptOut} />
      </section>

      <section className="card" aria-labelledby="dsar-h">
        <h2 id="dsar-h" style={{ marginBottom: 12 }}>Data subject requests</h2>
        <p className="muted" style={{ marginBottom: 14 }}>Exercise your rights under GDPR and CCPA/CPRA. We respond within the statutory window.</p>
        <DsarForm />
      </section>

      <section className="stack" aria-labelledby="dh-h">
        <h2 id="dh-h">Your requests ({requests.length})</h2>
        {requests.length === 0 ? (
          <div className="card empty"><h3>No requests filed</h3><p>Requests you submit appear here with their status.</p></div>
        ) : (
          <div className="table-wrap">
            <table className="data">
              <caption className="visually-hidden">Data subject requests you have filed</caption>
              <thead><tr><th scope="col">Type</th><th scope="col">Regime</th><th scope="col">Status</th><th scope="col">Filed</th><th scope="col">Due by</th></tr></thead>
              <tbody>
                {requests.map((r) => (
                  <tr key={r.id}>
                    <td>{r.type.replace(/_/g, ' ')}</td>
                    <td className="nowrap">{r.regime.toUpperCase()}</td>
                    <td><DsarStatusBadge status={r.status} /></td>
                    <td className="muted nowrap">{fmtDate(r.requestedAt)}</td>
                    <td className="muted nowrap">{fmtDate(r.slaDue)}</td>
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
