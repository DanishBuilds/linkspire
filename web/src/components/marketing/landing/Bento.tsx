'use client';

import Image from 'next/image';
import { useEffect, useMemo, useRef, useState } from 'react';
import { AnimatePresence, motion, useInView, useReducedMotion } from 'framer-motion';
import {
  Area, AreaChart, ResponsiveContainer, Tooltip, XAxis, YAxis,
} from 'recharts';
import QRCode from 'qrcode';
import { Check, ShieldCheck, Globe, Download } from 'lucide-react';
import { sdkCode } from '@/lib/landing-content';
import { Reveal } from './motion';

/* ---------------- Cell 1: live event stream ---------------- */
type Evt = { id: number; flag: string; city: string; browser: string; ref: string; code: string };
const SEED: Omit<Evt, 'id'>[] = [
  { flag: '🇺🇸', city: 'New York', browser: 'Chrome', ref: 'twitter.com', code: 'launch' },
  { flag: '🇩🇪', city: 'Berlin', browser: 'Firefox', ref: 'newsletter', code: 'docs' },
  { flag: '🇯🇵', city: 'Tokyo', browser: 'Safari', ref: 'google.com', code: 'pricing' },
  { flag: '🇬🇧', city: 'London', browser: 'Edge', ref: 'direct', code: 'webinar' },
  { flag: '🇧🇷', city: 'São Paulo', browser: 'Chrome', ref: 'linkedin.com', code: 'blog-ai' },
  { flag: '🇮🇳', city: 'Mumbai', browser: 'Chrome', ref: 'whatsapp', code: 'dl-ios' },
  { flag: '🇫🇷', city: 'Paris', browser: 'Safari', ref: 'reddit.com', code: 'promo' },
  { flag: '🇨🇦', city: 'Toronto', browser: 'Firefox', ref: 'direct', code: 'launch' },
];

function EventStream() {
  const reduce = useReducedMotion();
  const ref = useRef<HTMLDivElement>(null);
  const inView = useInView(ref, { margin: '-40px' });
  const [events, setEvents] = useState<Evt[]>(() => SEED.slice(0, 4).map((e, i) => ({ ...e, id: i })));
  const n = useRef(100);

  useEffect(() => {
    if (reduce || !inView) return;
    const t = setInterval(() => {
      const pick = SEED[Math.floor(Math.random() * SEED.length)]!;
      setEvents((prev) => [{ ...pick, id: n.current++ }, ...prev].slice(0, 5));
    }, 1600);
    return () => clearInterval(t);
  }, [reduce, inView]);

  return (
    <div ref={ref} className="mt-4 space-y-2" aria-live="polite" aria-label="Live click events">
      <AnimatePresence initial={false}>
        {events.map((e) => (
          <motion.div
            key={e.id}
            layout
            initial={reduce ? false : { opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.3 }}
            className="flex items-center gap-3 rounded-lg border border-border bg-[var(--surface-2)] px-3 py-2 text-sm"
          >
            <span className="text-base" aria-hidden>{e.flag}</span>
            <span className="w-20 shrink-0 text-text">{e.city}</span>
            <span className="hidden w-16 shrink-0 text-text-muted sm:inline">{e.browser}</span>
            <span className="hidden flex-1 truncate text-text-faint md:inline">{e.ref}</span>
            <span className="ml-auto font-mono text-xs text-accent">/{e.code}</span>
          </motion.div>
        ))}
      </AnimatePresence>
    </div>
  );
}

/* ---------------- Cell 2: privacy toggles ---------------- */
function Toggle({ label, on }: { label: string; on: boolean }) {
  return (
    <div className="flex items-center justify-between gap-3 py-2">
      <span className="text-sm text-text">{label}</span>
      <span
        className="relative h-6 w-11 shrink-0 rounded-full transition-colors duration-300"
        style={{ background: on ? 'var(--success)' : 'var(--border-strong)' }}
        role="img"
        aria-label={on ? 'on' : 'off'}
      >
        <motion.span
          className="absolute top-0.5 grid h-5 w-5 place-items-center rounded-full bg-white"
          animate={{ left: on ? 22 : 2 }}
          transition={{ type: 'spring', stiffness: 500, damping: 30 }}
        >
          {on && <Check className="h-3 w-3 text-success" />}
        </motion.span>
      </span>
    </div>
  );
}

function PrivacyToggles() {
  const reduce = useReducedMotion();
  const ref = useRef<HTMLDivElement>(null);
  const inView = useInView(ref, { once: true, margin: '-40px' });
  const [count, setCount] = useState(reduce ? 3 : 0);
  useEffect(() => {
    if (reduce || !inView) return;
    let i = 0;
    const t = setInterval(() => { i += 1; setCount(i); if (i >= 3) clearInterval(t); }, 350);
    return () => clearInterval(t);
  }, [inView, reduce]);
  const items = ['No visitor fingerprinting', 'No third-party cookies', 'GDPR compliant by design'];
  return (
    <div ref={ref} className="mt-4">
      <ShieldCheck className="h-8 w-8 text-accent" aria-hidden />
      <div className="mt-3 divide-y divide-border">
        {items.map((label, i) => <Toggle key={label} label={label} on={i < count} />)}
      </div>
      <p className="mt-3 text-sm text-text-muted">Your visitors&apos; data stays theirs.</p>
    </div>
  );
}

/* ---------------- Cell 3: custom domains ---------------- */
function DomainsMini() {
  const rows: [string, 'ok' | 'pending'][] = [
    ['links.yourco.com', 'ok'], ['go.yourco.com', 'ok'], ['try.yourco.com', 'pending'],
  ];
  return (
    <div className="mt-4 space-y-2">
      {rows.map(([d, s]) => (
        <div key={d} className="flex items-center justify-between gap-3 rounded-lg border border-border bg-[var(--surface-2)] px-3 py-2.5">
          <span className="flex items-center gap-2 font-mono text-sm text-text"><Globe className="h-4 w-4 text-text-faint" />{d}</span>
          {s === 'ok' ? (
            <span className="inline-flex items-center gap-1 rounded-full bg-[var(--success-soft)] px-2 py-0.5 text-xs font-medium text-success">
              <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-success" /> Verified
            </span>
          ) : (
            <span className="inline-flex items-center gap-1 rounded-full bg-[var(--warning-soft)] px-2 py-0.5 text-xs font-medium text-warning">
              <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-warning" /> Pending
            </span>
          )}
        </div>
      ))}
    </div>
  );
}

/* ---------------- Cell 4: analytics mini-dashboard ---------------- */
const SERIES = Array.from({ length: 30 }, (_, i) => ({
  d: i + 1,
  clicks: Math.round(300 + 260 * Math.sin(i / 4) + i * 34 + (i % 3) * 40),
}));
const FUNNEL = [
  { label: 'Clicks', pct: 100, value: '12.4K' },
  { label: 'QR scans', pct: 46, value: '5.7K' },
  { label: 'Conversions', pct: 18, value: '2.2K' },
];

function AnalyticsMini() {
  const reduce = useReducedMotion();
  return (
    <div className="mt-4">
      <div className="h-40 w-full">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={SERIES} margin={{ top: 4, right: 4, bottom: 0, left: 0 }}>
            <defs>
              <linearGradient id="clk" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="var(--c1)" stopOpacity={0.35} />
                <stop offset="100%" stopColor="var(--c1)" stopOpacity={0} />
              </linearGradient>
            </defs>
            <XAxis dataKey="d" hide />
            <YAxis hide />
            <Tooltip
              cursor={{ stroke: 'var(--border-strong)' }}
              contentStyle={{ background: 'var(--surface)', border: '1px solid var(--border-strong)', borderRadius: 10, fontSize: 12, color: 'var(--text)' }}
              labelFormatter={(d) => `Day ${d}`}
              formatter={(v: number) => [v.toLocaleString(), 'Clicks']}
            />
            <Area type="monotone" dataKey="clicks" stroke="var(--c1)" strokeWidth={2} fill="url(#clk)" isAnimationActive={!reduce} animationDuration={900} />
          </AreaChart>
        </ResponsiveContainer>
      </div>

      <div className="mt-4 grid grid-cols-3 gap-3">
        {[['Total clicks', '12.4K', '+18%'], ['Unique visitors', '8.2K', ''], ['Top country', 'US', '34%']].map(([k, v, delta]) => (
          <div key={k} className="rounded-lg border border-border bg-[var(--surface-2)] p-3">
            <div className="text-xs text-text-muted">{k}</div>
            <div className="mt-1 flex items-baseline gap-1.5">
              <span className="text-lg font-bold text-text">{v}</span>
              {delta && <span className="text-xs font-medium text-success">{delta}</span>}
            </div>
          </div>
        ))}
      </div>

      <div className="mt-4 space-y-1.5">
        {FUNNEL.map((f) => (
          <div key={f.label} className="flex items-center gap-3">
            <span className="w-24 shrink-0 text-xs text-text-muted">{f.label}</span>
            <div className="h-6 flex-1 overflow-hidden rounded-md bg-[var(--surface-2)]">
              <motion.div
                initial={reduce ? false : { width: 0 }}
                whileInView={{ width: `${f.pct}%` }}
                viewport={{ once: true }}
                transition={{ duration: 0.8, ease: 'easeOut' }}
                className="flex h-full items-center justify-end rounded-md pr-2 text-[11px] font-semibold text-white"
                style={{ background: 'linear-gradient(90deg, var(--c1), var(--c2))' }}
              >
                {f.value}
              </motion.div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

/* ---------------- Cell 5: QR customizer ---------------- */
function QrCustomizer() {
  const [color, setColor] = useState('#7c5cfc');
  const [corner, setCorner] = useState<'rounded' | 'square' | 'dots'>('rounded');
  const [png, setPng] = useState('');
  useEffect(() => {
    QRCode.toDataURL('https://lnk.to/linkspire', { margin: 1, width: 220, color: { dark: color, light: '#ffffff' } })
      .then(setPng).catch(() => {});
  }, [color]);

  async function download(fmt: 'png' | 'svg') {
    let href = png;
    if (fmt === 'svg') {
      const svg = await QRCode.toString('https://lnk.to/linkspire', { type: 'svg', margin: 1, color: { dark: color, light: '#ffffff' } });
      href = `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`;
    }
    const a = document.createElement('a');
    a.href = href; a.download = `linkspire-qr.${fmt}`; a.click();
  }

  return (
    <div className="mt-4 flex flex-col items-center gap-4 sm:flex-row sm:items-start">
      <div className="group [perspective:800px]">
        {png && (
          <Image
            src={png} alt="QR code preview" width={120} height={120}
            unoptimized
            className="rounded-xl bg-white p-2 shadow-card transition-transform duration-500 group-hover:[transform:rotateY(18deg)]"
            style={{ borderRadius: corner === 'square' ? 4 : corner === 'dots' ? 24 : 12 }}
          />
        )}
      </div>
      <div className="w-full flex-1 space-y-3">
        <label className="flex items-center justify-between gap-2 text-sm text-text-muted">
          Color
          <input type="color" value={color} onChange={(e) => setColor(e.target.value)} aria-label="QR color" className="h-7 w-10 cursor-pointer rounded border border-border bg-transparent" />
        </label>
        <div className="flex items-center justify-between gap-2 text-sm text-text-muted">
          Corners
          <div className="flex gap-1">
            {(['rounded', 'square', 'dots'] as const).map((c) => (
              <button key={c} type="button" onClick={() => setCorner(c)}
                className={`rounded-md px-2 py-1 text-xs capitalize ${corner === c ? 'bg-accent text-white' : 'border border-border text-text-muted hover:text-text'}`}>
                {c}
              </button>
            ))}
          </div>
        </div>
        <div className="flex gap-2 pt-1">
          <button type="button" onClick={() => download('png')} className="inline-flex flex-1 items-center justify-center gap-1.5 rounded-lg border border-border bg-[var(--surface-2)] px-3 py-2 text-sm text-text hover:border-accent">
            <Download className="h-4 w-4" /> PNG
          </button>
          <button type="button" onClick={() => download('svg')} className="inline-flex flex-1 items-center justify-center gap-1.5 rounded-lg border border-border bg-[var(--surface-2)] px-3 py-2 text-sm text-text hover:border-accent">
            <Download className="h-4 w-4" /> SVG
          </button>
        </div>
      </div>
    </div>
  );
}

/* ---------------- Cell 6: code block ---------------- */
function highlight(code: string) {
  return code.split('\n').map((line, i) => {
    const comment = line.trimStart().startsWith('//');
    return (
      <div key={i} className={comment ? 'text-text-faint' : 'text-text'}>
        {line || ' '}
      </div>
    );
  });
}

function CodeBlock() {
  return (
    <div className="mt-4">
      <pre className="overflow-x-auto rounded-xl border border-border bg-[var(--surface-2)] p-4 font-mono text-[13px] leading-relaxed">
        <code>{highlight(sdkCode)}</code>
      </pre>
      <div className="mt-3 flex flex-wrap gap-2">
        {['TypeScript SDK', 'REST API', 'Webhooks'].map((b) => (
          <span key={b} className="rounded-full border border-border bg-[var(--surface-2)] px-3 py-1 text-xs text-text-muted">{b}</span>
        ))}
      </div>
    </div>
  );
}

/* ---------------- Cell shell ---------------- */
function Cell({ span, title, desc, children }: { span: string; title: string; desc: string; children: React.ReactNode }) {
  return (
    <div className={`${span} rounded-2xl border border-border bg-elevated p-5 transition-colors duration-200 hover:border-accent`}>
      <h3 className="text-lg font-semibold text-text">{title}</h3>
      <p className="mt-1 text-sm text-text-muted">{desc}</p>
      {children}
    </div>
  );
}

export function Bento() {
  return (
    <section className="px-4 py-20">
      <div className="mx-auto max-w-content">
        <Reveal>
          <h2 className="text-center text-3xl font-bold tracking-tight text-text sm:text-4xl" style={{ letterSpacing: '-0.02em' }}>
            Everything you need. Nothing you don&apos;t.
          </h2>
          <p className="mx-auto mt-3 max-w-2xl text-center text-text-muted">
            Linkspire gives you the power of enterprise link management with the simplicity of a personal tool.
          </p>
        </Reveal>

        <Reveal delay={0.08}>
          <div className="mt-10 grid grid-cols-1 gap-4 md:grid-cols-12">
            <Cell span="md:col-span-8" title="Lightning-fast redirects" desc="Edge redirects with a live, privacy-safe event stream.">
              <EventStream />
            </Cell>
            <Cell span="md:col-span-4" title="Privacy by default" desc="No fingerprinting. No cross-site cookies.">
              <PrivacyToggles />
            </Cell>
            <Cell span="md:col-span-4" title="Custom domains" desc="Bring your own domain with guided DNS.">
              <DomainsMini />
            </Cell>
            <Cell span="md:col-span-8" title="Analytics that respect privacy" desc="Aggregates and per-day uniques, never raw IPs.">
              <AnalyticsMini />
            </Cell>
            <Cell span="md:col-span-6" title="QR codes" desc="Customizable, downloadable as PNG or SVG.">
              <QrCustomizer />
            </Cell>
            <Cell span="md:col-span-6" title="Developer API" desc="A typed SDK, REST, and signed webhooks.">
              <CodeBlock />
            </Cell>
          </div>
        </Reveal>
      </div>
    </section>
  );
}
