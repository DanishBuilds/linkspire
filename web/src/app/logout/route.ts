import { NextResponse } from 'next/server';
import { clearSession } from '@/lib/session';

// GET /logout — clears the session cookie then returns to sign-in. Used both by
// the nav logout and by the app layout when the API reports the token is invalid.
export async function GET(req: Request) {
  await clearSession();
  return NextResponse.redirect(new URL('/login', req.url));
}
