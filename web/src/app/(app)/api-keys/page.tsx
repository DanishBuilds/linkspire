import type { Metadata } from 'next';
import * as api from '@/lib/api';
import { ROLE_RANK } from '@/lib/types';
import { Badge } from '@/components/Badge';
import { Alert } from '@/components/Alert';
import { ConfirmSubmit } from '@/components/ConfirmSubmit';
import { CreateKeyForm } from '@/components/resources/CreateKeyForm';
import { revokeKeyAction } from '@/app/actions/resources';
import { fmtDate } from '@/lib/format';

export const metadata: Metadata = { title: 'API keys' };

export default async function ApiKeysPage() {
  const [keys, me] = await Promise.all([api.listKeys(), api.me()]);
  const role = me.workspaces[0]?.role ?? 'viewer';
  const canAdmin = ROLE_RANK[role] >= ROLE_RANK.admin;

  return (
    <div className="stack">
      <div className="page-head">
        <h1>API keys</h1>
        <p>Keys authenticate programmatic access with a bearer token. Secrets are shown once and stored only as a hash.</p>
      </div>

      {canAdmin ? (
        <section className="card" aria-labelledby="ck-h">
          <h2 id="ck-h" style={{ marginBottom: 14 }}>Create a key</h2>
          <CreateKeyForm />
        </section>
      ) : (
        <Alert variant="info" title="Admin access required">Managing API keys needs an admin role.</Alert>
      )}

      <section className="stack" aria-labelledby="kl-h">
        <h2 id="kl-h">Keys ({keys.length})</h2>
        {keys.length === 0 ? (
          <div className="card empty"><h3>No API keys</h3><p>Create one to use the Linkspire API.</p></div>
        ) : (
          <div className="table-wrap">
            <table className="data">
              <caption className="visually-hidden">API keys in this workspace</caption>
              <thead>
                <tr>
                  <th scope="col">Name</th><th scope="col">Prefix</th><th scope="col">Scopes</th>
                  <th scope="col">Status</th><th scope="col">Expires</th>
                  {canAdmin ? <th scope="col"><span className="visually-hidden">Actions</span></th> : null}
                </tr>
              </thead>
              <tbody>
                {keys.map((k) => (
                  <tr key={k.id}>
                    <td>{k.name}</td>
                    <td><code>{k.prefix}…</code></td>
                    <td>{k.scopes.length ? k.scopes.join(', ') : <span className="muted">default</span>}</td>
                    <td>{k.revokedAt ? <Badge tone="bad">revoked</Badge> : <Badge tone="ok">active</Badge>}</td>
                    <td className="muted nowrap">{fmtDate(k.expiresAt)}</td>
                    {canAdmin ? (
                      <td>
                        {k.revokedAt ? null : (
                          <form action={revokeKeyAction}>
                            <input type="hidden" name="id" value={k.id} />
                            <ConfirmSubmit confirm={`Revoke key “${k.name}”? Applications using it will stop working.`}>Revoke</ConfirmSubmit>
                          </form>
                        )}
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
