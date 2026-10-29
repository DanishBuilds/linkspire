'use client';

import { useEffect, useState } from 'react';
import { Icon } from '@/components/icons';

const STORAGE_KEY = 'linkspire-theme';
type Theme = 'light' | 'dark';

function readTheme(): Theme {
  if (typeof window === 'undefined') return 'light';
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === 'light' || stored === 'dark') return stored;
  } catch {
    /* private mode */
  }
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function applyTheme(theme: Theme) {
  document.documentElement.setAttribute('data-theme', theme);
  try {
    localStorage.setItem(STORAGE_KEY, theme);
  } catch {
    /* private mode */
  }
}

export function ThemeToggle() {
  const [theme, setTheme] = useState<Theme>('light');

  useEffect(() => {
    const initial = readTheme();
    applyTheme(initial);
    setTheme(initial);
  }, []);

  function toggle() {
    const next: Theme = theme === 'light' ? 'dark' : 'light';
    applyTheme(next);
    setTheme(next);
  }

  const label = theme === 'light' ? 'Switch to dark theme' : 'Switch to light theme';

  return (
    <button type="button" className="btn secondary small theme-toggle" onClick={toggle} aria-label={label} title={label}>
      {theme === 'light' ? <Icon.Moon width={18} height={18} /> : <Icon.Sun width={18} height={18} />}
      <span className="theme-toggle-label">{theme === 'light' ? 'Dark' : 'Light'}</span>
    </button>
  );
}
