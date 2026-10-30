'use client';

import { useEffect, useState } from 'react';
import { useTheme } from 'next-themes';
import { Icon } from '@/components/icons';

export function ThemeToggle() {
  const { resolvedTheme, setTheme } = useTheme();
  const [mounted, setMounted] = useState(false);
  useEffect(() => setMounted(true), []);

  const isDark = resolvedTheme === 'dark';
  const label = isDark ? 'Switch to light theme' : 'Switch to dark theme';

  // Render a stable shell until mounted to avoid hydration mismatch.
  return (
    <button
      type="button"
      className="btn secondary small theme-toggle"
      onClick={() => setTheme(isDark ? 'light' : 'dark')}
      aria-label={label}
      title={label}
      suppressHydrationWarning
    >
      {mounted ? (
        <>
          {isDark ? <Icon.Sun width={18} height={18} /> : <Icon.Moon width={18} height={18} />}
          <span className="theme-toggle-label">{isDark ? 'Light' : 'Dark'}</span>
        </>
      ) : (
        <span className="theme-toggle-label" style={{ opacity: 0 }}>Theme</span>
      )}
    </button>
  );
}
