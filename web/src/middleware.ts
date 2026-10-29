// src/middleware.ts — lightweight route gate. Presence of the session cookie
// decides routing only; token validity is enforced by the API (401 handling).
import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

const COOKIE = process.env.SESSION_COOKIE_NAME ?? 'linkspire_session';
const PROTECTED = ['/dashboard', '/links', '/domains', '/api-keys', '/webhooks', '/settings'];
const AUTH_PAGES = ['/login', '/register'];

export function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl;
  const hasSession = Boolean(req.cookies.get(COOKIE)?.value);

  if (PROTECTED.some((p) => pathname === p || pathname.startsWith(`${p}/`))) {
    if (!hasSession) {
      const url = req.nextUrl.clone();
      url.pathname = '/login';
      url.searchParams.set('next', pathname);
      return NextResponse.redirect(url);
    }
  }
  if (AUTH_PAGES.includes(pathname) && hasSession) {
    const url = req.nextUrl.clone();
    url.pathname = '/dashboard';
    url.search = '';
    return NextResponse.redirect(url);
  }
  return NextResponse.next();
}

export const config = {
  matcher: ['/dashboard/:path*', '/links/:path*', '/domains/:path*', '/api-keys/:path*', '/webhooks/:path*', '/settings/:path*', '/login', '/register'],
};
