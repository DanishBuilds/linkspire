'use client';

import Link from 'next/link';
import { useState } from 'react';
import Image from 'next/image';
import { AnimatePresence, motion, useReducedMotion } from 'framer-motion';
import QRCode from 'qrcode';
import { Check, Copy, Github, CornerDownLeft, Link2 } from 'lucide-react';
import { trustItems } from '@/lib/landing-content';

function randomCode() {
  const alphabet = 'abcdefghijkmnpqrstuvwxyz23456789';
  return Array.from({ length: 7 }, () => alphabet[Math.floor(Math.random() * alphabet.length)]).join('');
}

function InlineShortener() {
  const reduce = useReducedMotion();
  const [url, setUrl] = useState('');
  const [result, setResult] = useState<{ short: string; qr: string } | null>(null);
  const [busy, setBusy] = useState(false);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function shorten(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = url.trim();
    if (!trimmed) { setError('Paste a URL first.'); return; }
    try {
      // Lenient validation for the demo; the real app validates server-side.
      new URL(trimmed.includes('://') ? trimmed : `https://${trimmed}`);
    } catch {
      setError('That does not look like a URL.');
      return;
    }
    setError(null);
    setBusy(true);
    const short = `lnk.to/${randomCode()}`;
    const qr = await QRCode.toDataURL(`https://${short}`, { margin: 1, width: 160, color: { dark: '#0a0a0b', light: '#ffffff' } });
    setBusy(false);
    setResult({ short, qr });
    setCopied(false);
  }

  async function copy() {
    if (!result) return;
    try { await navigator.clipboard.writeText(`https://${result.short}`); setCopied(true); setTimeout(() => setCopied(false), 1600); } catch { /* */ }
  }

  return (
    <div className="mx-auto mt-10 w-full max-w-2xl">
      <form onSubmit={shorten} className="flex flex-col gap-2 sm:flex-row">
        <div className="relative flex-1">
          <Link2 className="pointer-events-none absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-text-faint" aria-hidden />
          <input
            type="text"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="Paste your long URL here..."
            aria-label="Long URL to shorten"
            className="w-full rounded-xl border border-border bg-[var(--surface-2)] py-3 pl-10 pr-4 text-text placeholder:text-text-faint outline-none focus:border-accent focus:ring-2 focus:ring-[var(--accent-glow)]"
          />
        </div>
        <button
          type="submit"
          disabled={busy}
          className="inline-flex items-center justify-center gap-2 rounded-xl px-6 py-3 font-semibold text-white shadow-[0_8px_24px_-8px_var(--accent-glow)] transition-transform hover:-translate-y-px disabled:opacity-70"
          style={{ background: 'var(--accent)' }}
        >
          {busy ? 'Shortening…' : (<>Shorten <CornerDownLeft className="h-4 w-4" /></>)}
        </button>
      </form>
      {error && <p className="mt-2 text-left text-sm text-danger">{error}</p>}

      <AnimatePresence>
        {result && (
          <motion.div
            initial={reduce ? false : { opacity: 0, y: 12, scale: 0.98 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 8 }}
            transition={{ type: 'spring', stiffness: 260, damping: 22 }}
            className="mt-3 flex items-center gap-4 rounded-xl border border-border bg-elevated p-4 text-left shadow-card"
          >
            <Image src={result.qr} alt="QR code preview" width={64} height={64} className="rounded-md bg-white p-1" unoptimized />
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <span className="truncate font-mono font-semibold text-accent">{result.short}</span>
                <button type="button" onClick={copy} aria-label="Copy short link" className="rounded-md p-1 text-text-muted hover:bg-[var(--surface-2)] hover:text-text">
                  {copied ? <Check className="h-4 w-4 text-success" /> : <Copy className="h-4 w-4" />}
                </button>
              </div>
              <div className="mt-1 flex items-center gap-3 text-xs text-text-muted">
                <span><strong className="text-text">0</strong> clicks</span>
                <span aria-hidden>·</span>
                <Link href="/register" className="text-accent hover:underline">Sign up to track this link</Link>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

export function Hero() {
  const reduce = useReducedMotion();
  return (
    <section className="relative overflow-hidden px-4 pt-16 pb-20 sm:pt-24">
      {/* subtle gradient glow */}
      <div
        aria-hidden
        className="pointer-events-none absolute left-1/2 top-0 -z-10 h-[520px] w-[900px] max-w-none -translate-x-1/2 opacity-[0.10] blur-3xl"
        style={{ background: 'linear-gradient(135deg, #7c5cfc 0%, #3b82f6 50%, #06b6d4 100%)' }}
      />
      <div className="mx-auto max-w-content text-center">
        <motion.h1
          initial={reduce ? false : { opacity: 0, y: 18 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, ease: [0.22, 1, 0.36, 1] }}
          className="mx-auto max-w-3xl text-4xl font-bold tracking-tight text-text sm:text-6xl"
          style={{ letterSpacing: '-0.03em' }}
        >
          Shorten links. Own your data.
        </motion.h1>
        <motion.p
          initial={reduce ? false : { opacity: 0, y: 18 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.08, ease: [0.22, 1, 0.36, 1] }}
          className="mx-auto mt-5 max-w-2xl text-lg text-text-muted"
        >
          The open-source link shortener with built-in analytics, QR codes, and zero tracking of your
          visitors. Self-host it or use our cloud.
        </motion.p>

        <motion.div
          initial={reduce ? false : { opacity: 0, y: 18 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.16, ease: [0.22, 1, 0.36, 1] }}
          className="mt-8 flex flex-wrap items-center justify-center gap-3"
        >
          <Link
            href="/register"
            className="rounded-xl px-6 py-3 font-semibold text-white shadow-[0_8px_30px_-8px_var(--accent-glow)] transition-transform hover:-translate-y-px"
            style={{ background: 'var(--accent)' }}
          >
            Start free
          </Link>
          <a
            href="https://github.com"
            className="inline-flex items-center gap-2 rounded-xl border border-border-strong bg-elevated px-6 py-3 font-semibold text-text transition-colors hover:border-accent"
          >
            <Github className="h-5 w-5" /> View on GitHub
          </a>
        </motion.div>

        <ul className="mt-6 flex flex-wrap items-center justify-center gap-x-5 gap-y-2 text-sm text-text-muted">
          {trustItems.map((t) => (
            <li key={t} className="inline-flex items-center gap-1.5">
              <Check className="h-4 w-4 text-success" aria-hidden /> {t}
            </li>
          ))}
        </ul>

        <InlineShortener />
      </div>
    </section>
  );
}
