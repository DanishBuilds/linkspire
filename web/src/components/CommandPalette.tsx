'use client';

import { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import * as Dialog from '@radix-ui/react-dialog';
import { Search, CornerDownLeft } from 'lucide-react';

type Cmd = { label: string; href: string; group: string };
const COMMANDS: Cmd[] = [
  { label: 'Overview', href: '/dashboard', group: 'Navigate' },
  { label: 'Links', href: '/links', group: 'Navigate' },
  { label: 'Domains', href: '/domains', group: 'Navigate' },
  { label: 'Analytics', href: '/insights', group: 'Navigate' },
  { label: 'API keys', href: '/api-keys', group: 'Navigate' },
  { label: 'Webhooks', href: '/webhooks', group: 'Navigate' },
  { label: 'Settings', href: '/settings', group: 'Navigate' },
  { label: 'Create a link', href: '/links', group: 'Actions' },
];

export function CommandPalette() {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [q, setQ] = useState('');
  const [active, setActive] = useState(0);

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setOpen((o) => !o);
      }
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  const results = useMemo(() => {
    const needle = q.trim().toLowerCase();
    return needle ? COMMANDS.filter((c) => c.label.toLowerCase().includes(needle)) : COMMANDS;
  }, [q]);

  function go(c: Cmd | undefined) {
    if (!c) return;
    setOpen(false); setQ('');
    router.push(c.href);
  }

  return (
    <Dialog.Root open={open} onOpenChange={(o) => { setOpen(o); if (!o) { setQ(''); setActive(0); } }}>
      <Dialog.Trigger asChild>
        <button type="button" className="cmdk-trigger" aria-label="Search (Command or Control + K)">
          <Search className="h-4 w-4" />
          <span className="cmdk-trigger-label">Search</span>
          <kbd className="cmdk-kbd">⌘K</kbd>
        </button>
      </Dialog.Trigger>
      <Dialog.Portal>
        <Dialog.Overlay className="cmdk-overlay" />
        <Dialog.Content className="cmdk" aria-label="Command menu">
          <Dialog.Title className="visually-hidden">Command menu</Dialog.Title>
          <div className="cmdk-input-row">
            <Search className="h-4 w-4 text-[var(--text-faint)]" aria-hidden />
            <input
              autoFocus
              value={q}
              onChange={(e) => { setQ(e.target.value); setActive(0); }}
              onKeyDown={(e) => {
                if (e.key === 'ArrowDown') { e.preventDefault(); setActive((a) => Math.min(results.length - 1, a + 1)); }
                else if (e.key === 'ArrowUp') { e.preventDefault(); setActive((a) => Math.max(0, a - 1)); }
                else if (e.key === 'Enter') { e.preventDefault(); go(results[active]); }
              }}
              placeholder="Search or jump to…"
              aria-label="Search commands"
              className="cmdk-input"
            />
          </div>
          <ul className="cmdk-list">
            {results.length === 0 ? (
              <li className="cmdk-empty">No matches.</li>
            ) : results.map((c, i) => (
              <li key={c.label}>
                <button
                  type="button"
                  className={`cmdk-item ${i === active ? 'active' : ''}`}
                  onMouseEnter={() => setActive(i)}
                  onClick={() => go(c)}
                >
                  <span className="cmdk-item-group">{c.group}</span>
                  <span>{c.label}</span>
                  {i === active ? <CornerDownLeft className="ml-auto h-3.5 w-3.5 opacity-60" /> : null}
                </button>
              </li>
            ))}
          </ul>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
