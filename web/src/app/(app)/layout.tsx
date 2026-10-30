import Link from 'next/link';
import { redirect } from 'next/navigation';
import * as api from '@/lib/api';
import { ApiError } from '@/lib/api';
import { NavLink } from '@/components/NavLink';
import { Icon } from '@/components/icons';
import { Badge } from '@/components/Badge';
import { logoutAction } from '@/app/actions/auth';
import { getWorkspaceId } from '@/lib/session';
import { WorkspaceSwitcher } from '@/components/WorkspaceSwitcher';
import { ThemeToggle } from '@/components/ThemeToggle';
import { CommandPalette } from '@/components/CommandPalette';
import type { Me } from '@/lib/types';

// Soft display cap for the sidebar usage meter (Free tier). The authoritative
// limit will come from the billing service; this is a UI affordance.
const PLAN_CAP = 10000;

export default async function AppLayout({ children }: { children: React.ReactNode }) {
  let me: Me;
  try {
    me = await api.me();
  } catch (e) {
    if (e instanceof ApiError && (e.status === 401 || e.status === 0)) redirect('/logout');
    throw e;
  }
  const linkCount = await api.listLinks(100).then((l) => l.length).catch(() => 0);
  const usagePct = Math.min(100, Math.round((linkCount / PLAN_CAP) * 100));

  const wsCookie = await getWorkspaceId();
  const selected = me.workspaces.find((w) => String(w.workspaceId) === wsCookie) ?? me.workspaces[0];

  return (
    <>
      <a href="#main" className="skip-link">Skip to content</a>
      <div className="shell">
        <aside className="sidebar" aria-label="Primary">
          <Link href="/" className="brand"><span className="brand-mark" aria-hidden="true" />Linkspire</Link>
          <nav className="nav" aria-label="Main navigation">
            <NavLink href="/dashboard"><Icon.Dashboard className="ico" />Overview</NavLink>
            <NavLink href="/links"><Icon.Link className="ico" />Links</NavLink>
            <NavLink href="/domains"><Icon.Globe className="ico" />Domains</NavLink>
            <NavLink href="/insights"><Icon.Chart className="ico" />Analytics</NavLink>
            <div className="nav-section" role="presentation">Developer</div>
            <NavLink href="/api-keys"><Icon.Key className="ico" />API keys</NavLink>
            <NavLink href="/webhooks"><Icon.Webhook className="ico" />Webhooks</NavLink>
            <div className="nav-section" role="presentation">Account</div>
            <NavLink href="/settings"><Icon.Settings className="ico" />Settings</NavLink>
            {me.isStaff ? (
              <>
                <div className="nav-section" role="presentation">Admin</div>
                <NavLink href="/admin/abuse"><Icon.Shield className="ico" />Abuse review</NavLink>
              </>
            ) : null}
          </nav>

          <div className="usage-meter" aria-label="Plan usage">
            <div className="usage-top">
              <span>Links used</span>
              <span className="usage-count">{linkCount.toLocaleString()}{linkCount >= 100 ? '+' : ''} / {PLAN_CAP.toLocaleString()}</span>
            </div>
            <div className="usage-track" role="progressbar" aria-valuenow={usagePct} aria-valuemin={0} aria-valuemax={100}>
              <span className="usage-fill" style={{ width: `${Math.max(usagePct, 2)}%` }} />
            </div>
          </div>

          <div className="sidebar-foot" aria-label="Product information">
            <Link href="/features">Features</Link>
            <Link href="/analytics">Analytics</Link>
            <Link href="/privacy">Privacy</Link>
            <Link href="/developers">Developers</Link>
          </div>
        </aside>

        <div className="main">
          <header className="topbar">
            <div className="cluster">
              <WorkspaceSwitcher workspaces={me.workspaces} currentId={selected?.workspaceId ?? 0} />
              {selected ? <Badge tone="info">{selected.role}</Badge> : null}
              {selected ? <span className="muted nowrap">{selected.homeRegion.toUpperCase()} region</span> : null}
            </div>
            <div className="cluster">
              <CommandPalette />
              <ThemeToggle />
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
