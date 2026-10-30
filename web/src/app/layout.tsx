import type { Metadata } from 'next';
import { headers } from 'next/headers';
import { fontMono, fontSans } from '@/lib/fonts';
import { ThemeProvider } from '@/components/ThemeProvider';
import './globals.css';

export const metadata: Metadata = {
  title: { default: 'Linkspire', template: '%s · Linkspire' },
  description: 'Privacy-first URL shortener dashboard.',
  robots: { index: false, follow: false }, // control plane: never indexed
  icons: { icon: '/favicon.svg' },
};

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const nonce = (await headers()).get('x-nonce') ?? undefined;
  return (
    <html lang="en" className={`${fontSans.variable} ${fontMono.variable}`} suppressHydrationWarning>
      <body className={fontSans.className}>
        <ThemeProvider nonce={nonce}>{children}</ThemeProvider>
      </body>
    </html>
  );
}
