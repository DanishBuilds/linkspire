import Link from 'next/link';
import { MarketingShell } from '@/components/marketing/MarketingShell';

type Props = {
  title: string;
  lead: string;
  children: React.ReactNode;
};

export function MktDocPage({ title, lead, children }: Props) {
  return (
    <MarketingShell>
      <article className="mkt-doc">
        <p className="mkt-doc-back"><Link href="/">← Back to home</Link></p>
        <header className="mkt-doc-head">
          <h1>{title}</h1>
          <p className="lead">{lead}</p>
        </header>
        <div className="mkt-doc-body">{children}</div>
      </article>
    </MarketingShell>
  );
}
