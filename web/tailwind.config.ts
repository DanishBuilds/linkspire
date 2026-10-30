import type { Config } from 'tailwindcss';

/**
 * Tailwind is wired to the CSS-variable design tokens defined in globals.css.
 * Dark mode is driven by the existing data-theme attribute (managed by
 * next-themes), so utilities and the hand-written token CSS stay in sync.
 */
const config: Config = {
  darkMode: ['class', '[data-theme="dark"]'],
  content: ['./src/**/*.{ts,tsx,mdx}'],
  theme: {
    extend: {
      colors: {
        base: 'var(--bg)',
        elevated: 'var(--surface)',
        surface: 'var(--surface-2)',
        border: 'var(--border)',
        'border-strong': 'var(--border-strong)',
        text: {
          DEFAULT: 'var(--text)',
          muted: 'var(--text-muted)',
          faint: 'var(--text-faint)',
        },
        accent: {
          DEFAULT: 'var(--accent)',
          hover: 'var(--accent-hover)',
          contrast: 'var(--accent-contrast)',
          soft: 'var(--accent-soft)',
        },
        success: { DEFAULT: 'var(--success)', soft: 'var(--success-soft)' },
        warning: { DEFAULT: 'var(--warning)', soft: 'var(--warning-soft)' },
        danger: { DEFAULT: 'var(--danger)', soft: 'var(--danger-soft)' },
        info: { DEFAULT: 'var(--info)', soft: 'var(--info-soft)' },
        c1: 'var(--c1)', c2: 'var(--c2)', c3: 'var(--c3)',
        c4: 'var(--c4)', c5: 'var(--c5)', c6: 'var(--c6)',
      },
      fontFamily: {
        sans: ['var(--font-sans)'],
        mono: ['var(--font-mono)'],
      },
      borderRadius: {
        sm: 'var(--radius-sm)',
        DEFAULT: 'var(--radius)',
        lg: 'var(--radius-lg)',
        xl: 'var(--radius-xl)',
        '2xl': 'var(--radius-2xl)',
      },
      maxWidth: { content: 'var(--maxw)' },
      boxShadow: {
        card: 'var(--shadow)',
        glow: '0 0 0 1px var(--accent-soft), 0 8px 40px -8px var(--accent-glow)',
      },
      keyframes: {
        'fade-up': {
          from: { opacity: '0', transform: 'translateY(16px)' },
          to: { opacity: '1', transform: 'translateY(0)' },
        },
        shimmer: {
          '100%': { transform: 'translateX(100%)' },
        },
        marquee: {
          from: { transform: 'translateX(0)' },
          to: { transform: 'translateX(-50%)' },
        },
      },
      animation: {
        'fade-up': 'fade-up 0.5s ease forwards',
        marquee: 'marquee 28s linear infinite',
      },
    },
  },
  corePlugins: { preflight: false },
  plugins: [],
};
export default config;
