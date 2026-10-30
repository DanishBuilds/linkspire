'use client';

import { Link2, Zap, ShieldCheck, Timer } from 'lucide-react';
import { logos, stats } from '@/lib/landing-content';
import { Reveal, Counter } from './motion';

const statIcons = [Link2, Zap, ShieldCheck, Timer];

export function Social() {
  return (
    <section className="px-4 py-16" aria-label="Social proof">
      <div className="mx-auto max-w-content">
        <Reveal>
          <p className="text-center text-sm font-medium uppercase tracking-wide text-text-faint">
            Trusted by developers and teams worldwide
          </p>
          <ul className="mt-8 grid grid-cols-2 gap-x-6 gap-y-6 sm:grid-cols-3 md:grid-cols-4">
            {logos.map((name) => (
              <li
                key={name}
                className="flex items-center justify-center text-lg font-semibold text-text-faint opacity-70 grayscale transition hover:text-text hover:opacity-100 hover:grayscale-0"
              >
                {name}
              </li>
            ))}
          </ul>
        </Reveal>

        <Reveal delay={0.1}>
          <dl className="mt-14 grid grid-cols-2 gap-6 md:grid-cols-4">
            {stats.map((s, i) => {
              const Ico = statIcons[i]!;
              return (
                <div key={s.label} className="text-center">
                  <Ico className="mx-auto mb-2 h-5 w-5 text-accent" aria-hidden />
                  <dd className="text-3xl font-bold tracking-tight text-text sm:text-4xl" style={{ letterSpacing: '-0.02em' }}>
                    <Counter to={s.value} decimals={s.decimals ?? 0} suffix={s.suffix} />
                  </dd>
                  <dt className="mt-1 text-sm text-text-muted">{s.label}</dt>
                </div>
              );
            })}
          </dl>
        </Reveal>
      </div>
    </section>
  );
}
