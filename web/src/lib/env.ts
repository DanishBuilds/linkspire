// src/lib/env.ts — server-side environment access. These values must never be
// sent to the browser (no NEXT_PUBLIC_ prefix except the short-link base URL,
// which is display-only).
function cleanBaseUrl(raw: string | undefined, fallback: string): string {
  // Vercel env set via PowerShell `echo` may include \r\n — strip all whitespace.
  return (raw?.trim() || fallback).replace(/\s+/g, '').replace(/\/+$/, '');
}

export const env = {
  apiUrl: cleanBaseUrl(process.env.LINKSPIRE_API_URL, 'http://localhost:8081'),
  shortBaseUrl: cleanBaseUrl(process.env.NEXT_PUBLIC_SHORT_BASE_URL, 'https://lnk.linkspire.io'),
  sessionCookie: process.env.SESSION_COOKIE_NAME ?? 'linkspire_session',
  cookieSecure: process.env.SESSION_COOKIE_SECURE === 'true',
};
