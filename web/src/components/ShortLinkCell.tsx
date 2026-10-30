import Link from 'next/link';
import { CopyButton } from './CopyButton';

// Short-code chip linking to the link detail, with an inline icon copy button.
export function ShortLinkCell({
  domainId, code, shortUrl,
}: { domainId: number; code: string; shortUrl: string }) {
  return (
    <div className="short-cell">
      <Link href={`/links/${domainId}/${encodeURIComponent(code)}`} className="code">{code}</Link>
      <CopyButton value={shortUrl} label="Copy short link" compact />
    </div>
  );
}
