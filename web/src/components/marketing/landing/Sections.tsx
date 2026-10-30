'use client';

import Link from 'next/link';
import { Check, Github, Star, Twitter } from 'lucide-react';
import { comparisons, testimonials, pricing } from '@/lib/landing-content';
import { Reveal } from './motion';

export function Comparison() {
  return (
    <section className="px-4 py-16">
      <div className="mx-auto max-w-content">
        <Reveal>
          <h2 className="text-center text-3xl font-bold tracking-tight text-text" style={{ letterSpacing: '-0.02em' }}>Why Linkspire?</h2>
          <div className="mt-10 grid grid-cols-1 gap-4 md:grid-cols-3">
            {comparisons.map((c) => (
              <div key={c.name} className="rounded-2xl border border-border bg-elevated p-6">
                <div className="text-sm font-semibold uppercase tracking-wide text-text-faint">{c.name}</div>
                <ul className="mt-4 space-y-3">
                  {c.points.map((p) => (
                    <li key={p} className="flex items-start gap-2 text-sm text-text">
                      <Check className="mt-0.5 h-4 w-4 shrink-0 text-success" aria-hidden /> {p}
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </Reveal>
      </div>
    </section>
  );
}

export function Testimonials() {
  return (
    <section className="px-4 py-16">
      <div className="mx-auto max-w-content">
        <Reveal>
          <h2 className="text-center text-3xl font-bold tracking-tight text-text" style={{ letterSpacing: '-0.02em' }}>Loved by builders</h2>
        </Reveal>
        <Reveal delay={0.08}>
          <div className="mt-10 gap-4 [column-fill:_balance] sm:columns-2 lg:columns-3">
            {testimonials.map((t) => (
              <figure key={t.name} className="mb-4 break-inside-avoid rounded-2xl border border-border bg-elevated p-5">
                {t.tweet && <Twitter className="mb-2 h-4 w-4 text-accent" aria-hidden />}
                <blockquote className={t.featured ? 'text-lg text-text' : 'text-sm text-text'}>
                  {t.highlight ? (
                    <>
                      {t.quote.split(t.highlight)[0]}
                      <strong className="font-semibold text-text">{t.highlight}</strong>
                      {t.quote.split(t.highlight)[1]}
                    </>
                  ) : t.quote}
                </blockquote>
                <figcaption className="mt-4 flex items-center gap-3">
                  <span className="grid h-9 w-9 place-items-center rounded-full text-sm font-semibold text-white" style={{ background: 'linear-gradient(135deg, var(--c1), var(--c4))' }}>{t.initials}</span>
                  <span className="text-sm">
                    <span className="block font-medium text-text">{t.name}</span>
                    <span className="block text-text-muted">{t.role}</span>
                  </span>
                </figcaption>
              </figure>
            ))}
          </div>
        </Reveal>
      </div>
    </section>
  );
}

export function OpenSource() {
  return (
    <section id="open-source" className="px-4 py-16">
      <div className="mx-auto max-w-3xl rounded-2xl border border-border bg-elevated p-8 text-center sm:p-12">
        <Reveal>
          <Github className="mx-auto h-10 w-10 text-text" aria-hidden />
          <h2 className="mt-4 text-3xl font-bold tracking-tight text-text" style={{ letterSpacing: '-0.02em' }}>Built in the open</h2>
          <p className="mx-auto mt-3 max-w-xl text-text-muted">
            Linkspire is open-source and self-hostable. Star us on GitHub, contribute, or deploy your own
            instance in minutes.
          </p>
          <div className="mt-6 flex flex-wrap items-center justify-center gap-3">
            <a href="https://github.com" className="inline-flex items-center gap-2 rounded-xl border border-border-strong bg-[var(--surface-2)] px-5 py-2.5 font-semibold text-text hover:border-accent">
              <Star className="h-4 w-4" /> Star on GitHub
              <span className="ml-1 rounded-md bg-elevated px-2 py-0.5 text-xs text-text-muted">8.4k</span>
            </a>
            <a href="https://github.com" className="inline-flex items-center gap-2 rounded-xl px-5 py-2.5 font-semibold text-white" style={{ background: 'var(--accent)' }}>
              Deploy with Docker
            </a>
          </div>
          <div className="mx-auto mt-6 max-w-md overflow-hidden rounded-xl border border-border bg-[var(--surface-2)] text-left">
            <div className="flex items-center gap-1.5 border-b border-border px-3 py-2">
              <span className="h-2.5 w-2.5 rounded-full bg-danger" /><span className="h-2.5 w-2.5 rounded-full bg-warning" /><span className="h-2.5 w-2.5 rounded-full bg-success" />
            </div>
            <pre className="px-4 py-3 font-mono text-[13px] leading-relaxed">
<span className="text-text-muted">$</span> <span className="text-text">docker run -d -p 3000:3000 linkspire/linkspire</span>{'\n'}<span className="text-success">✓ Linkspire is running at http://localhost:3000</span>
            </pre>
          </div>
        </Reveal>
      </div>
    </section>
  );
}

export function Pricing() {
  return (
    <section id="pricing" className="px-4 py-16">
      <div className="mx-auto max-w-content">
        <Reveal>
          <h2 className="text-center text-3xl font-bold tracking-tight text-text" style={{ letterSpacing: '-0.02em' }}>Simple, honest pricing</h2>
          <div className="mt-10 grid grid-cols-1 gap-4 md:grid-cols-3">
            {pricing.map((p) => (
              <div
                key={p.name}
                className="relative flex flex-col rounded-2xl border bg-elevated p-6"
                style={p.popular
                  ? { borderColor: 'var(--accent)', boxShadow: '0 0 0 1px var(--accent), 0 16px 50px -20px var(--accent-glow)' }
                  : { borderColor: 'var(--border)' }}
              >
                {p.popular && (
                  <span className="absolute -top-3 left-1/2 -translate-x-1/2 rounded-full px-3 py-1 text-xs font-semibold text-white" style={{ background: 'var(--accent)' }}>
                    Most popular
                  </span>
                )}
                <div className="text-sm font-semibold text-text">{p.name}</div>
                <div className="mt-2 flex items-baseline gap-1">
                  <span className="text-3xl font-bold text-text">{p.price}</span>
                  <span className="text-sm text-text-muted">/ {p.cadence}</span>
                </div>
                <ul className="mt-5 flex-1 space-y-2.5">
                  {p.features.map((f) => (
                    <li key={f} className="flex items-start gap-2 text-sm text-text-muted">
                      <Check className="mt-0.5 h-4 w-4 shrink-0 text-success" aria-hidden /> {f}
                    </li>
                  ))}
                </ul>
                <Link
                  href="/register"
                  className={`mt-6 rounded-xl px-4 py-2.5 text-center text-sm font-semibold transition-colors ${p.popular ? 'text-white' : 'border border-border-strong text-text hover:border-accent'}`}
                  style={p.popular ? { background: 'var(--accent)' } : undefined}
                >
                  {p.cta}
                </Link>
              </div>
            ))}
          </div>
          <p className="mt-6 text-center text-sm text-text-muted">
            Or <Link href="/developers" className="text-accent hover:underline">self-host for free, forever</Link>.
          </p>
        </Reveal>
      </div>
    </section>
  );
}

export function FinalCTA() {
  return (
    <section className="relative overflow-hidden px-4 py-24 text-center">
      <div aria-hidden className="pointer-events-none absolute left-1/2 top-1/2 -z-10 h-80 w-[800px] max-w-none -translate-x-1/2 -translate-y-1/2 opacity-[0.1] blur-3xl" style={{ background: 'linear-gradient(135deg, #7c5cfc, #06b6d4)' }} />
      <Reveal>
        <h2 className="mx-auto max-w-2xl text-4xl font-bold tracking-tight text-text sm:text-5xl" style={{ letterSpacing: '-0.03em' }}>
          Start shortening. Start owning.
        </h2>
        <p className="mx-auto mt-4 max-w-xl text-text-muted">
          Join thousands of developers and teams who chose privacy-first link management.
        </p>
        <Link href="/register" className="mt-8 inline-block rounded-xl px-7 py-3.5 text-lg font-semibold text-white shadow-[0_10px_40px_-10px_var(--accent-glow)] transition-transform hover:-translate-y-px" style={{ background: 'var(--accent)' }}>
          Create your first link, free
        </Link>
        <p className="mt-4 text-sm text-text-muted">No credit card · No tracking · No compromise</p>
      </Reveal>
    </section>
  );
}
