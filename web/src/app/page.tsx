import type { Metadata } from 'next';
import { Nav } from '@/components/marketing/landing/Nav';
import { Hero } from '@/components/marketing/landing/Hero';
import { Social } from '@/components/marketing/landing/Social';
import { Bento } from '@/components/marketing/landing/Bento';
import { Comparison, Testimonials, OpenSource, Pricing, FinalCTA } from '@/components/marketing/landing/Sections';
import { Footer } from '@/components/marketing/landing/Footer';

export const metadata: Metadata = {
  title: 'Linkspire: privacy-first link management',
  description:
    'The open-source link shortener with built-in analytics, QR codes, and zero tracking of your visitors. Self-host it or use our cloud.',
  robots: { index: true, follow: true },
  openGraph: {
    title: 'Linkspire: shorten links, own your data',
    description: 'Privacy-first, open-source URL shortener with honest analytics and QR codes.',
    type: 'website',
  },
};

export default function HomePage() {
  return (
    <div className="min-h-screen bg-base font-sans text-text">
      <Nav />
      <main>
        <Hero />
        <Social />
        <Bento />
        <Comparison />
        <Testimonials />
        <OpenSource />
        <Pricing />
        <FinalCTA />
      </main>
      <Footer />
    </div>
  );
}
