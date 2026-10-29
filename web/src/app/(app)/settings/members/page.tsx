import type { Metadata } from 'next';
import * as api from '@/lib/api';
import { ROLE_RANK } from '@/lib/types';
import { getWorkspaceId } from '@/lib/session';
import { Badge } from '@/components/Badge';
import { Alert } from '@/components/Alert';
import { RoleForm } from '@/components/account/RoleForm';

export const metadata: Metadata = { title: 'Members' };

export default async function MembersPage() {
  const me = await api.me();
  const wsCookie = await getWorkspaceId();
  const selected = me.workspaces.find((w) => String(w.workspaceId) === wsCookie) ?? me.workspaces[0];
  const role = selected?.role ?? 'viewer';
  const canAdmin = ROLE_RANK[role] >= ROLE_RANK.admin;
  const canOwner = ROLE_RANK[role] >= ROLE_RANK.owner;

  if (!selected) return <Alert variant="warning">No workspace selected.</Alert>;
  if (!canAdmin) return <Alert variant="info" title="Admin access required">Viewing workspace members needs an admin role.</Alert>;

  const members = await api.listMembers(selected.workspaceId);

  return (
    <section className="stack" aria-labelledby="mem-h">
      <div>
        <h2 id="mem-h">Members of {selected.workspaceName || `workspace #${selected.workspaceId}`}</h2>
        <p className="muted" style={{ marginTop: 4 }}>
          Roles follow the hierarchy viewer, member, admin, owner. Only owners can change roles.
        </p>
      </div>
      <div className="table-wrap">
        <table className="data">
          <caption className="visually-hidden">Workspace members and their roles</caption>
          <thead>
            <tr><th scope="col">User</th><th scope="col">Role</th>{canOwner ? <th scope="col">Change role</th> : null}</tr>
          </thead>
          <tbody>
            {members.map((m) => (
              <tr key={m.userId}>
                <td>User #{m.userId}{m.userId === me.id ? <> <span className="muted">(you)</span></> : null}</td>
                <td><Badge tone="info">{m.role}</Badge></td>
                {canOwner ? (
                  <td>{m.userId === me.id ? <span className="muted">—</span> : <RoleForm workspaceId={selected.workspaceId} userId={m.userId} currentRole={m.role} />}</td>
                ) : null}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p className="muted" style={{ fontSize: '0.88rem' }}>
        Member emails are not exposed by the API for privacy; members are shown by id.
      </p>
    </section>
  );
}
