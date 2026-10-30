'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import { Github, Twitter, MessageCircle } from 'lucide-react';
import { useReducedMotion } from 'framer-motion';
import { footerColumns } from '@/lib/landing-content';

function LiveStat({ label, start, step }: { label: string; start: number; step: number }) {
  const reduce = useReducedMotion();
  const [n, setN] = useState(start);
  useEffect(() => {
    if (reduce) return;
    const t = setInterval(() => setN((v) => v + Math.ceil(Math.random() * step)), 1500);
    return () => clearInterval(t);
  }, [reduce, step]);
  return (
    <span className="tabular-nums">
      <span className="text-text-muted">{label}: </span>
      <strong className="font-semibold text-text">{n.toLocaleString('en-US')}</strong>
    </span>
  );
}

export function Footer() {
  return (
    <footer className="border-t border-border bg-elevated">
      <div className="mx-auto grid max-w-content grid-cols-2 gap-8 px-4 py-12 md:grid-cols-4">
        <div className="col-span-2 md:col-span-1">
          <Link href="/" className="flex items-center gap-2 font-bold text-text">
            <span className="grid h-7 w-7 place-items-center rounded-lg text-white" style={{ background: 'linear-gradient(135deg, var(--c1), var(--c2))' }} aria-hidden>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round"><path d="M9 15 15 9"/><path d="M11 6l1-1a4 4 0 0 1 6 6l-1 1"/><path d="M13 18l-1 1a4 4 0 0 1-6-6l1-1"/></svg>
            </span>
            Linkspire
          </Link>
          <p className="mt-3 max-w-xs text-sm text-text-muted">Privacy-first link management. Shorten, track, and own your data.</p>
          <div className="mt-4 flex gap-3 text-text-muted">
            <a href="https://github.com" aria-label="GitHub" className="hover:text-text"><Github className="h-5 w-5" /></a>
            <a href="https://x.com" aria-label="X" className="hover:text-text"><Twitter className="h-5 w-5" /></a>
            <a href="https://discord.com" aria-label="Discord" className="hover:text-text"><MessageCircle className="h-5 w-5" /></a>
          </div>
        </div>
        {footerColumns.map((col) => (
          <div key={col.title}>
            <h3 className="text-sm font-semibold text-text">{col.title}</h3>
            <ul className="mt-3 space-y-2">
              {col.links.map(([label, href]) => (
                <li key={label}><Link href={href} className="text-sm text-text-muted hover:text-text">{label}</Link></li>
              ))}
            </ul>
          </div>
        ))}
      </div>

      <div className="border-t border-border">
        <div className="mx-auto flex max-w-content flex-wrap items-center justify-between gap-3 px-4 py-4 text-sm text-text-muted">
          <span>© 2026 Linkspire</span>
          <div className="flex flex-wrap items-center gap-x-5 gap-y-1">
            {['SOC 2', 'GDPR', 'CCPA'].map((b) => (
              <span key={b} className="rounded-md border border-border px-2 py-0.5 text-xs">{b}</span>
            ))}
          </div>
        </div>
      </div>

      <div className="border-t border-border" style={{ background: 'color-mix(in srgb, var(--accent) 6%, var(--bg))' }}>
        <div className="mx-auto flex max-w-content flex-wrap items-center justify-center gap-x-6 gap-y-1 px-4 py-3 text-sm">
          <LiveStat label="Links created" start={50213487} step={7} />
          <span aria-hidden className="text-text-faint">·</span>
          <LiveStat label="Redirects served" start={2104887340} step={240} />
          <span aria-hidden className="text-text-faint">·</span>
          <span className="text-text-muted">Avg latency: <strong className="font-semibold text-text">15ms</strong></span>
        </div>
      </div>
    </footer>
  );
}
