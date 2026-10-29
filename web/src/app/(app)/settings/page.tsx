import type { Metadata } from 'next';
import * as api from '@/lib/api';
import { Badge } from '@/components/Badge';

export const metadata: Metadata = { title: 'Profile' };

export default async function ProfilePage() {
  const me = await api.me();
  return (
    <div className="stack">
      <section className="card" aria-labelledby="p-h">
        <h2 id="p-h">Profile</h2>
        <dl className="grid cols-2" style={{ marginTop: 12 }}>
          <div><dt className="muted">Email</dt><dd>{me.email}</dd></div>
          <div><dt className="muted">Home region</dt><dd>{me.homeRegion.toUpperCase()}</dd></div>
          <div><dt className="muted">Two-factor auth</dt><dd>{me.mfaEnabled ? <Badge tone="ok">enabled</Badge> : <Badge tone="warn">off</Badge>}</dd></div>
          <div><dt className="muted">Do Not Sell / Share</dt><dd>{me.gpcOptOut ? <Badge tone="ok">on</Badge> : <Badge>off</Badge>}</dd></div>
        </dl>
      </section>

      <section className="card" aria-labelledby="ws-h">
        <h2 id="ws-h">Workspaces</h2>
        <div className="table-wrap" style={{ marginTop: 12 }}>
          <table className="data">
            <caption className="visually-hidden">Workspaces you belong to</caption>
            <thead><tr><th scope="col">Workspace</th><th scope="col">Role</th><th scope="col">Region</th></tr></thead>
            <tbody>
              {me.workspaces.map((w) => (
                <tr key={w.workspaceId}>
                  <td>{w.workspaceName || `#${w.workspaceId}`}</td>
                  <td><Badge tone="info">{w.role}</Badge></td>
                  <td className="muted">{w.homeRegion.toUpperCase()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
