import { JetBrains_Mono, Plus_Jakarta_Sans } from 'next/font/google';

/** Primary UI and marketing typeface — crisp at small sizes, strong display rhythm. */
export const fontSans = Plus_Jakarta_Sans({
  subsets: ['latin'],
  variable: '--font-plus-jakarta',
  display: 'swap',
  weight: ['400', '500', '600', '700', '800'],
});

/** Code blocks, short URLs, API examples. */
export const fontMono = JetBrains_Mono({
  subsets: ['latin'],
  variable: '--font-jetbrains-mono',
  display: 'swap',
  weight: ['400', '500', '600'],
});
