// src/middleware.ts — session gate + nonce-based CSP (Next.js 15 applies the nonce to
// framework inline scripts when x-nonce is present on the request).
import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

const COOKIE = process.env.SESSION_COOKIE_NAME ?? 'linkspire_session';
const PROTECTED = ['/dashboard', '/links', '/domains', '/api-keys', '/webhooks', '/settings', '/admin'];
const AUTH_PAGES = ['/login', '/register'];

function csp(nonce: string): string {
  const dev = process.env.NODE_ENV === 'development';
  return [
    "default-src 'self'",
    "base-uri 'self'",
    "form-action 'self'",
    "frame-ancestors 'none'",
    "object-src 'none'",
    "img-src 'self' data: blob:",
    "style-src 'self' 'unsafe-inline'",
    `script-src 'self' 'nonce-${nonce}' 'strict-dynamic'${dev ? " 'unsafe-eval'" : ''}`,
    "connect-src 'self'",
    "font-src 'self'",
  ].join('; ');
}

function withSecurityHeaders(res: NextResponse, nonce: string): NextResponse {
  res.headers.set('Content-Security-Policy', csp(nonce));
  res.headers.set('X-Content-Type-Options', 'nosniff');
  res.headers.set('Referrer-Policy', 'no-referrer');
  res.headers.set('X-Frame-Options', 'DENY');
  res.headers.set('Permissions-Policy', 'camera=(), microphone=(), geolocation=()');
  if (process.env.LINKSPIRE_NO_CACHE === 'true') {
    res.headers.set('Cache-Control', 'no-store, must-revalidate');
  }
  return res;
}

export function middleware(req: NextRequest) {
  const nonce = Buffer.from(crypto.randomUUID()).toString('base64');
  const requestHeaders = new Headers(req.headers);
  requestHeaders.set('x-nonce', nonce);

  const { pathname } = req.nextUrl;
  const hasSession = Boolean(req.cookies.get(COOKIE)?.value);

  if (PROTECTED.some((p) => pathname === p || pathname.startsWith(`${p}/`))) {
    if (!hasSession) {
      const url = req.nextUrl.clone();
      url.pathname = '/login';
      url.searchParams.set('next', pathname);
      return withSecurityHeaders(NextResponse.redirect(url), nonce);
    }
  }
  if (AUTH_PAGES.includes(pathname) && hasSession) {
    const url = req.nextUrl.clone();
    url.pathname = '/dashboard';
    url.search = '';
    return withSecurityHeaders(NextResponse.redirect(url), nonce);
  }

  return withSecurityHeaders(
    NextResponse.next({ request: { headers: requestHeaders } }),
    nonce,
  );
}

export const config = {
  matcher: [
    {
      source: '/((?!_next/static|_next/image|favicon.ico|.*\\.(?:svg|png|jpg|jpeg|gif|webp|ico)$).*)',
      missing: [
        { type: 'header', key: 'next-router-prefetch' },
        { type: 'header', key: 'purpose', value: 'prefetch' },
      ],
    },
  ],
};
