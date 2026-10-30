// Geist Sans + Geist Mono, self-hosted by the `geist` package (no network
// fetch at build time). Exposes CSS variables consumed by globals.css:
//   --font-geist-sans, --font-geist-mono
import { GeistSans } from 'geist/font/sans';
import { GeistMono } from 'geist/font/mono';

export const fontSans = GeistSans;
export const fontMono = GeistMono;
