'use client';
import Link from 'next/link';
import { usePathname } from 'next/navigation';

const tabs = [
  { href: '/settings', label: 'Profile', exact: true },
  { href: '/settings/security', label: 'Security' },
  { href: '/settings/privacy', label: 'Privacy' },
];

export function SettingsTabs() {
  const pathname = usePathname();
  return (
    <nav className="tabs" aria-label="Settings sections">
      {tabs.map((t) => {
        const active = t.exact ? pathname === t.href : pathname.startsWith(t.href);
        return <Link key={t.href} href={t.href} aria-current={active ? 'page' : undefined}>{t.label}</Link>;
      })}
    </nav>
  );
}
