import Image from 'next/image';
import { Icon } from '@/components/icons';
import type { Link } from '@/lib/types';

// QR code for a link. Generated on demand by /api/qr: shown inline and offered
// as instant SVG/PNG downloads. No background worker or object storage needed.
export function QrPanel({ link }: { link: Link }) {
  const base = `/api/qr/${link.domainId}/${encodeURIComponent(link.code)}`;
  return (
    <section className="card link-qr-card" id="qr-code" aria-labelledby="qr-h">
      <h2 id="qr-h" style={{ margin: 0 }}><Icon.Qr /> QR code</h2>
      <p className="muted" style={{ marginTop: 6, maxWidth: '52ch' }}>
        A scannable image of <span className="mono">{link.shortUrl}</span>. Point a phone camera at it to open the link.
      </p>

      <div style={{ display: 'flex', gap: 20, alignItems: 'center', flexWrap: 'wrap', marginTop: 14 }}>
        <Image
          src={`${base}?format=svg`}
          alt={`QR code for ${link.shortUrl}`}
          width={160}
          height={160}
          unoptimized
          style={{ border: '1px solid var(--border)', borderRadius: 12, background: '#fff', padding: 8 }}
        />
        <div className="cluster" style={{ gap: 10 }}>
          <a className="btn" href={`${base}?format=png&download=1`} download>
            <Icon.External />Download PNG
          </a>
          <a className="btn secondary" href={`${base}?format=svg&download=1`} download>
            <Icon.External />Download SVG
          </a>
        </div>
      </div>
    </section>
  );
}
