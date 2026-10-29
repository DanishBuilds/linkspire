import type { Metadata } from 'next';
import * as api from '@/lib/api';
import { ROLE_RANK } from '@/lib/types';
import { Badge } from '@/components/Badge';
import { Alert } from '@/components/Alert';
import { ConfirmSubmit } from '@/components/ConfirmSubmit';
import { CreateWebhookForm } from '@/components/resources/CreateWebhookForm';
import { deleteWebhookAction } from '@/app/actions/resources';

export const metadata: Metadata = { title: 'Webhooks' };

export default async function WebhooksPage() {
  const [hooks, me] = await Promise.all([api.listWebhooks(), api.me()]);
  const role = me.workspaces[0]?.role ?? 'viewer';
  const canAdmin = ROLE_RANK[role] >= ROLE_RANK.admin;

  return (
    <div className="stack">
      <div className="page-head">
        <h1>Webhooks</h1>
        <p>Receive signed HTTPS callbacks when links are created or flagged and when domains verify.</p>
      </div>

      {canAdmin ? (
        <section className="card" aria-labelledby="cw-h">
          <h2 id="cw-h" style={{ marginBottom: 14 }}>Register a webhook</h2>
          <CreateWebhookForm />
        </section>
      ) : (
        <Alert variant="info" title="Admin access required">Managing webhooks needs an admin role.</Alert>
      )}

      <section className="stack" aria-labelledby="wl-h">
        <h2 id="wl-h">Endpoints ({hooks.length})</h2>
        {hooks.length === 0 ? (
          <div className="card empty"><h3>No webhooks</h3><p>Register an endpoint to receive events.</p></div>
        ) : (
          <div className="table-wrap">
            <table className="data">
              <caption className="visually-hidden">Registered webhook endpoints</caption>
              <thead>
                <tr>
                  <th scope="col">URL</th><th scope="col">Events</th><th scope="col">Status</th>
                  {canAdmin ? <th scope="col"><span className="visually-hidden">Actions</span></th> : null}
                </tr>
              </thead>
              <tbody>
                {hooks.map((h) => (
                  <tr key={h.id}>
                    <td><span className="truncate" title={h.url}>{h.url}</span></td>
                    <td>{h.events.map((e) => <Badge key={e}>{e}</Badge>)}</td>
                    <td>{h.active ? <Badge tone="ok">active</Badge> : <Badge tone="warn">paused</Badge>}</td>
                    {canAdmin ? (
                      <td>
                        <form action={deleteWebhookAction}>
                          <input type="hidden" name="id" value={h.id} />
                          <ConfirmSubmit confirm="Delete this webhook endpoint?">Delete</ConfirmSubmit>
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
