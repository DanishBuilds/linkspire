'use client';

import Link from 'next/link';
import { useState } from 'react';
import { Sparkles, X, Menu, Github } from 'lucide-react';
import { ThemeToggle } from '@/components/ThemeToggle';
import { navLinks } from '@/lib/landing-content';

export function Nav() {
  const [announce, setAnnounce] = useState(true);
  const [open, setOpen] = useState(false);

  return (
    <header className="sticky top-0 z-50">
      {announce && (
        <div
          className="relative text-sm text-text border-b border-border"
          style={{ background: 'color-mix(in srgb, var(--accent) 12%, var(--bg))' }}
        >
          <div className="mx-auto flex max-w-content items-center justify-center gap-2 px-4 py-2 pr-10">
            <Sparkles className="h-4 w-4 text-accent" aria-hidden />
            <span className="text-center">
              v2.0 is live: privacy analytics, QR codes, and more.{' '}
              <Link href="/features" className="font-semibold text-accent hover:underline">See what&apos;s new →</Link>
            </span>
          </div>
          <button
            type="button"
            onClick={() => setAnnounce(false)}
            aria-label="Dismiss announcement"
            className="absolute right-3 top-1/2 -translate-y-1/2 rounded-md p-1 text-text-muted hover:text-text"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      )}

      <div
        className="border-b border-border"
        style={{
          background: 'color-mix(in srgb, var(--bg) 80%, transparent)',
          backdropFilter: 'blur(12px)',
          WebkitBackdropFilter: 'blur(12px)',
        }}
      >
        <nav className="mx-auto flex max-w-content items-center justify-between gap-6 px-4 py-3" aria-label="Primary">
          <Link href="/" className="flex items-center gap-2 font-bold text-text">
            <span
              className="grid h-8 w-8 place-items-center rounded-lg text-white"
              style={{ background: 'linear-gradient(135deg, var(--c1), var(--c2))' }}
              aria-hidden
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round"><path d="M9 15 15 9"/><path d="M11 6l1-1a4 4 0 0 1 6 6l-1 1"/><path d="M13 18l-1 1a4 4 0 0 1-6-6l1-1"/></svg>
            </span>
            Linkspire
          </Link>

          <div className="hidden items-center gap-1 md:flex">
            {navLinks.map((l) => (
              <Link key={l.href} href={l.href} className="rounded-md px-3 py-2 text-sm text-text-muted transition-colors hover:bg-elevated hover:text-text">
                {l.label}
              </Link>
            ))}
          </div>

          <div className="flex items-center gap-2">
            <ThemeToggle />
            <Link href="/login" className="hidden rounded-md px-3 py-2 text-sm font-medium text-text-muted hover:text-text sm:block">
              Log in
            </Link>
            <Link
              href="/register"
              className="rounded-lg px-4 py-2 text-sm font-semibold text-white shadow-[0_6px_20px_-6px_var(--accent-glow)] transition-transform hover:-translate-y-px"
              style={{ background: 'var(--accent)' }}
            >
              Get started
            </Link>
            <button
              type="button"
              className="rounded-md p-2 text-text md:hidden"
              aria-label="Toggle menu"
              aria-expanded={open}
              onClick={() => setOpen((o) => !o)}
            >
              {open ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
            </button>
          </div>
        </nav>

        {open && (
          <div className="border-t border-border px-4 py-3 md:hidden">
            <div className="flex flex-col gap-1">
              {navLinks.map((l) => (
                <Link key={l.href} href={l.href} onClick={() => setOpen(false)} className="rounded-md px-3 py-2 text-sm text-text-muted hover:bg-elevated hover:text-text">
                  {l.label}
                </Link>
              ))}
              <a href="https://github.com" className="mt-1 flex items-center gap-2 rounded-md px-3 py-2 text-sm text-text-muted hover:bg-elevated hover:text-text">
                <Github className="h-4 w-4" /> GitHub
              </a>
            </div>
          </div>
        )}
      </div>
    </header>
  );
}
