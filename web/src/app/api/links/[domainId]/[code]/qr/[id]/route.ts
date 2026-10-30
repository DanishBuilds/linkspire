import { env } from '@/lib/env';
import { getToken, getWorkspaceId } from '@/lib/session';

export async function GET(
  _req: Request,
  ctx: { params: Promise<{ domainId: string; code: string; id: string }> },
) {
  const { domainId, code, id } = await ctx.params;
  const token = await getToken();
  if (!token) return new Response('Unauthorized', { status: 401 });

  const headers: Record<string, string> = { Authorization: `Bearer ${token}` };
  const ws = await getWorkspaceId();
  if (ws) headers['X-Workspace-Id'] = ws;

  const upstream = await fetch(
    `${env.apiUrl}/v1/links/${domainId}/${encodeURIComponent(code)}/qr/${id}/file`,
    { headers, cache: 'no-store' },
  );
  if (!upstream.ok) {
    const text = await upstream.text();
    return new Response(text || upstream.statusText, { status: upstream.status });
  }

  const out = new Headers();
  const ct = upstream.headers.get('content-type');
  const cd = upstream.headers.get('content-disposition');
  if (ct) out.set('Content-Type', ct);
  if (cd) out.set('Content-Disposition', cd);
  return new Response(upstream.body, { status: 200, headers: out });
}
