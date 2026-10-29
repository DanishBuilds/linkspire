import { redirect } from 'next/navigation';
import * as api from '@/lib/api';
import { ApiError } from '@/lib/api';
import { NavLink } from '@/components/NavLink';
import { Icon } from '@/components/icons';
import { Badge } from '@/components/Badge';
import { logoutAction } from '@/app/actions/auth';
import type { Me } from '@/lib/types';

export default async function AppLayout({ children }: { children: React.ReactNode }) {
  let me: Me;
  try {
    me = await api.me();
  } catch (e) {
    if (e instanceof ApiError && (e.status === 401 || e.status === 0)) redirect('/logout');
    throw e;
  }
  const primary = me.workspaces[0];

  return (
    <>
      <a href="#main" className="skip-link">Skip to content</a>
      <div className="shell">
        <aside className="sidebar" aria-label="Primary">
          <div className="brand"><span className="brand-mark" aria-hidden="true" />Linkspire</div>
          <nav className="nav" aria-label="Main navigation">
            <NavLink href="/dashboard"><Icon.Dashboard className="ico" />Overview</NavLink>
            <NavLink href="/links"><Icon.Link className="ico" />Links</NavLink>
            <NavLink href="/domains"><Icon.Globe className="ico" />Domains</NavLink>
            <div className="nav-section" role="presentation">Developer</div>
            <NavLink href="/api-keys"><Icon.Key className="ico" />API keys</NavLink>
            <NavLink href="/webhooks"><Icon.Webhook className="ico" />Webhooks</NavLink>
            <div className="nav-section" role="presentation">Account</div>
            <NavLink href="/settings"><Icon.Settings className="ico" />Settings</NavLink>
          </nav>
        </aside>

        <div className="main">
          <header className="topbar">
            <div className="cluster">
              <strong>{primary?.workspaceName || 'Workspace'}</strong>
              {primary ? <Badge tone="info">{primary.role}</Badge> : null}
              {primary ? <span className="muted nowrap">{primary.homeRegion.toUpperCase()} region</span> : null}
            </div>
            <div className="cluster">
              <span className="muted truncate" title={me.email}>{me.email}</span>
              <form action={logoutAction}>
                <button type="submit" className="btn secondary small"><Icon.Logout />Sign out</button>
              </form>
            </div>
          </header>
          <main id="main" className="content">{children}</main>
        </div>
      </div>
    </>
  );
}
