import { SettingsTabs } from '@/components/account/SettingsTabs';

export default function SettingsLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="stack">
      <div className="page-head"><h1>Settings</h1></div>
      <SettingsTabs />
      {children}
    </div>
  );
}
