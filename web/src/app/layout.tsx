import type { Metadata } from 'next';
import Script from 'next/script';
import { fontMono, fontSans } from '@/lib/fonts';
import './globals.css';

export const metadata: Metadata = {
  title: { default: 'Linkspire', template: '%s · Linkspire' },
  description: 'Privacy-first URL shortener dashboard.',
  robots: { index: false, follow: false }, // control plane: never indexed
  icons: { icon: '/favicon.svg' },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${fontSans.variable} ${fontMono.variable}`} suppressHydrationWarning>
      <body className={fontSans.className}>
        <Script src="/theme-init.js" strategy="beforeInteractive" />
        {children}
      </body>
    </html>
  );
}
