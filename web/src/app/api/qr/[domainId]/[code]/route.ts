import QRCode from 'qrcode';
import * as api from '@/lib/api';
import { env } from '@/lib/env';
import { getToken } from '@/lib/session';

// On-demand QR generation. Renders a scannable image of the link's short URL
// directly (no worker, no object storage), so it always works and is instant.
//   /api/qr/{domainId}/{code}?format=svg|png&download=1
export async function GET(
  _req: Request,
  ctx: { params: Promise<{ domainId: string; code: string }> },
) {
  if (!(await getToken())) return new Response('Unauthorized', { status: 401 });

  const { domainId, code } = await ctx.params;
  const url = new URL(_req.url);
  const format = url.searchParams.get('format') === 'png' ? 'png' : 'svg';
  const download = url.searchParams.get('download') === '1';

  // Authoritative short URL (handles custom domains); fall back to the configured base.
  let target = `${env.shortBaseUrl}/${code}`;
  try {
    const link = await api.getLink(Number(domainId), code);
    if (link.shortUrl) target = link.shortUrl;
  } catch {
    /* fall back to constructed URL */
  }

  const filename = `linkspire-${code}.${format}`;
  const disposition = `${download ? 'attachment' : 'inline'}; filename="${filename}"`;
  const opts = { margin: 2, width: 512, errorCorrectionLevel: 'M' as const };

  try {
    if (format === 'png') {
      const buf = await QRCode.toBuffer(target, { ...opts, type: 'png' });
      return new Response(new Uint8Array(buf), {
        headers: {
          'Content-Type': 'image/png',
          'Content-Disposition': disposition,
          'Cache-Control': 'private, max-age=300',
        },
      });
    }
    const svg = await QRCode.toString(target, { ...opts, type: 'svg' });
    return new Response(svg, {
      headers: {
        'Content-Type': 'image/svg+xml; charset=utf-8',
        'Content-Disposition': disposition,
        'Cache-Control': 'private, max-age=300',
      },
    });
  } catch {
    return new Response('Could not generate QR code', { status: 500 });
  }
}
