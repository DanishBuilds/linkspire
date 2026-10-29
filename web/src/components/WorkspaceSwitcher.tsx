'use client';
import { useRef } from 'react';
import { switchWorkspaceAction } from '@/app/actions/workspace';
import type { Membership } from '@/lib/types';

// Workspace picker in the app header. Auto-submits on change. When the user
// belongs to a single workspace it renders as plain text, not a control.
export function WorkspaceSwitcher({ workspaces, currentId }: { workspaces: Membership[]; currentId: number }) {
  const ref = useRef<HTMLFormElement>(null);
  if (workspaces.length <= 1) {
    return <strong>{workspaces[0]?.workspaceName || 'Workspace'}</strong>;
  }
  return (
    <form action={switchWorkspaceAction} ref={ref}>
      <label htmlFor="ws-switch" className="visually-hidden">Switch workspace</label>
      <select
        id="ws-switch" name="workspace_id" className="select"
        defaultValue={String(currentId)}
        style={{ minHeight: 36, paddingTop: 4, paddingBottom: 4, fontWeight: 600, maxWidth: 220 }}
        onChange={() => ref.current?.requestSubmit()}
      >
        {workspaces.map((w) => (
          <option key={w.workspaceId} value={w.workspaceId}>
            {w.workspaceName || `Workspace #${w.workspaceId}`}
          </option>
        ))}
      </select>
    </form>
  );
}
