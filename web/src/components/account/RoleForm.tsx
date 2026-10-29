'use client';
import { useActionState } from 'react';
import { changeRoleAction, type MemberState } from '@/app/actions/members';
import { SubmitButton } from '@/components/SubmitButton';

const initial: MemberState = {};

// Inline role editor for one member (owner only).
export function RoleForm({ workspaceId, userId, currentRole }: { workspaceId: number; userId: number; currentRole: string }) {
  const [, action] = useActionState(changeRoleAction, initial);
  return (
    <form action={action} className="cluster" style={{ gap: 6 }}>
      <input type="hidden" name="workspace_id" value={workspaceId} />
      <input type="hidden" name="user_id" value={userId} />
      <label htmlFor={`role-${userId}`} className="visually-hidden">Role for user {userId}</label>
      <select id={`role-${userId}`} name="role" className="select" defaultValue={currentRole}
        style={{ minHeight: 34, paddingTop: 2, paddingBottom: 2, width: 'auto' }}>
        <option value="owner">owner</option>
        <option value="admin">admin</option>
        <option value="member">member</option>
        <option value="viewer">viewer</option>
      </select>
      <SubmitButton className="btn secondary small" pendingLabel="Saving…">Save</SubmitButton>
    </form>
  );
}
