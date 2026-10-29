// src/lib/env.ts — server-side environment access. These values must never be
// sent to the browser (no NEXT_PUBLIC_ prefix except the short-link base URL,
// which is display-only).
export const env = {
  apiUrl: (process.env.LINKSPIRE_API_URL ?? 'http://localhost:8081').replace(/\/+$/, ''),
  shortBaseUrl: (process.env.NEXT_PUBLIC_SHORT_BASE_URL ?? 'https://lnk.linkspire.io').replace(/\/+$/, ''),
  sessionCookie: process.env.SESSION_COOKIE_NAME ?? 'linkspire_session',
  cookieSecure: process.env.SESSION_COOKIE_SECURE === 'true',
};
