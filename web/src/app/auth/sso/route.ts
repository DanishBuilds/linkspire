import { NextResponse } from 'next/server';
import { env } from '@/lib/env';

// Kicks off OIDC at the API (keeping LINKSPIRE_API_URL server-side). The API
// redirects to the provider, then back to OIDC_POST_LOGIN_URL (= /auth/oidc).
export async function GET() {
  return NextResponse.redirect(`${env.apiUrl}/v1/auth/oidc/start`);
}
