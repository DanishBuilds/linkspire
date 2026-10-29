import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: { default: 'Linkspire', template: '%s · Linkspire' },
  description: 'Privacy-first URL shortener dashboard.',
  robots: { index: false, follow: false }, // control plane: never indexed
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
