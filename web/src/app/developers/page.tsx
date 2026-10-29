import type { Metadata } from 'next';
import Link from 'next/link';
import { MktDocPage } from '@/components/marketing/MktDocPage';
import { apiExample } from '@/lib/marketing-content';

export const metadata: Metadata = {
  title: 'Developers',
  description: 'REST API, scoped keys, idempotency, and signed webhooks for Linkspire.',
  robots: { index: true, follow: true },
};

export default function DevelopersPage() {
  return (
    <MktDocPage
      title="Build on Linkspire"
      lead="A versioned REST API with scoped keys, idempotency, and signed webhooks. Automate link creation, domain setup, and event-driven workflows."
    >
      <h2>Authentication</h2>
      <p>
        Use a bearer token from the dashboard (API keys) or a session cookie for browser flows. Keys are scoped to
        workspaces and can be limited to read-only or specific resources.
      </p>

      <h2>Create a link</h2>
      <pre className="code-block" aria-label="Example API request">{apiExample}</pre>

      <h2>Idempotency</h2>
      <p>
        Send <code>Idempotency-Key</code> on mutating requests. Retries with the same key return the original response
        without creating duplicate links.
      </p>

      <h2>Webhooks</h2>
      <ul>
        <li>Subscribe to link and domain lifecycle events from the dashboard</li>
        <li>Each delivery is signed (HMAC) so you can verify authenticity</li>
        <li>Automatic retries with backoff; dead-letter visibility in the control plane</li>
      </ul>

      <h2>Rate limits and errors</h2>
      <p>
        Responses use standard HTTP status codes and RFC 7807 problem details for errors. Rate limits are returned in
        headers; burst capacity is sized for campaign traffic.
      </p>

      <h2>OpenAPI</h2>
      <p>
        The management API schema lives in the repository as OpenAPI 3. Import it into Postman or generate clients for
        your stack. Base URL in production: <code>https://api.linkspire.io/v1</code> (local Docker: port 8081).
      </p>

      <p className="mkt-doc-cta">
        <Link href="/register" className="btn">Get an API key</Link>
        <Link href="/login" className="btn secondary">Open the dashboard</Link>
      </p>
    </MktDocPage>
  );
}
