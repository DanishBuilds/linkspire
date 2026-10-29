import { NextResponse } from 'next/server';
import { setSession } from '@/lib/session';

// Receives the Linkspire JWT the OIDC callback placed in the URL fragment and
// stores it as the httpOnly session cookie.
export async function POST(req: Request) {
  let token = '';
  try {
    const body = await req.json();
    token = typeof body?.token === 'string' ? body.token : '';
  } catch {
    token = '';
  }
  if (!token || token.split('.').length !== 3) {
    return NextResponse.json({ error: 'invalid token' }, { status: 400 });
  }
  await setSession(token);
  return NextResponse.json({ ok: true });
}
