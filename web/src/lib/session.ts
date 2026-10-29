// src/lib/session.ts — httpOnly session cookie helpers.
// The API JWT lives ONLY in an httpOnly, SameSite=Lax cookie set by the Next.js
// server. The browser never reads or stores the token (XSS cannot exfiltrate it).
import { cookies } from 'next/headers';
import { env } from './env';

const MAX_AGE = 12 * 60 * 60; // mirrors the API TOKEN_TTL default (12h)

export async function getToken(): Promise<string | null> {
  const store = await cookies();
  return store.get(env.sessionCookie)?.value ?? null;
}

export async function setSession(token: string): Promise<void> {
  const store = await cookies();
  store.set(env.sessionCookie, token, {
    httpOnly: true,
    sameSite: 'lax',
    secure: env.cookieSecure,
    path: '/',
    maxAge: MAX_AGE,
  });
}

export async function clearSession(): Promise<void> {
  const store = await cookies();
  store.delete(env.sessionCookie);
}

const WS_COOKIE = `${env.sessionCookie}_ws`;

// The selected workspace id (sent as X-Workspace-Id). Optional: when unset the
// API falls back to the user's first membership.
export async function getWorkspaceId(): Promise<string | null> {
  const store = await cookies();
  return store.get(WS_COOKIE)?.value ?? null;
}

export async function setWorkspaceId(id: string): Promise<void> {
  const store = await cookies();
  store.set(WS_COOKIE, id, {
    httpOnly: true, sameSite: 'lax', secure: env.cookieSecure, path: '/', maxAge: 12 * 60 * 60,
  });
}

export async function clearWorkspaceId(): Promise<void> {
  const store = await cookies();
  store.delete(WS_COOKIE);
}

