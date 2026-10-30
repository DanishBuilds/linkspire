import { env } from '@/lib/env';

/** Public probe: dashboard can show whether Vercel can reach the laptop API. */
export async function GET() {
  try {
    const res = await fetch(`${env.apiUrl}/healthz`, { cache: 'no-store' });
    const body = await res.text();
    return Response.json({
      ok: res.ok,
      apiUrl: env.apiUrl,
      status: res.status,
      body: body.slice(0, 80),
    });
  } catch (e) {
    return Response.json(
      {
        ok: false,
        apiUrl: env.apiUrl,
        error: e instanceof Error ? e.message : 'fetch failed',
      },
      { status: 503 },
    );
  }
}
