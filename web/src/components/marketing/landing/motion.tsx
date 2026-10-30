'use client';

import { motion, useInView, useReducedMotion, animate } from 'framer-motion';
import { useEffect, useRef, useState, type ReactNode } from 'react';

/** One orchestrated fade-up per section on scroll-in. Respects reduced motion. */
export function Reveal({
  children, delay = 0, className, as = 'div',
}: { children: ReactNode; delay?: number; className?: string; as?: 'div' | 'section' }) {
  const reduce = useReducedMotion();
  const MotionTag = as === 'section' ? motion.section : motion.div;
  return (
    <MotionTag
      className={className}
      initial={reduce ? false : { opacity: 0, y: 16 }}
      whileInView={reduce ? undefined : { opacity: 1, y: 0 }}
      viewport={{ once: true, margin: '-80px' }}
      transition={{ duration: 0.5, delay, ease: [0.22, 1, 0.36, 1] }}
    >
      {children}
    </MotionTag>
  );
}

/** Count-up number, triggered once on scroll-in. */
export function Counter({
  to, decimals = 0, suffix = '', className,
}: { to: number; decimals?: number; suffix?: string; className?: string }) {
  const ref = useRef<HTMLSpanElement>(null);
  const inView = useInView(ref, { once: true, margin: '-60px' });
  const reduce = useReducedMotion();
  const [val, setVal] = useState(0);

  useEffect(() => {
    if (!inView) return;
    if (reduce) { setVal(to); return; }
    const controls = animate(0, to, {
      duration: 1.1,
      ease: 'easeOut',
      onUpdate: (v) => setVal(v),
    });
    return () => controls.stop();
  }, [inView, to, reduce]);

  const text = val.toLocaleString('en-US', {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  });
  return <span ref={ref} className={className}>{text}{suffix}</span>;
}
