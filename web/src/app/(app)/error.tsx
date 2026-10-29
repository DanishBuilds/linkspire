'use client';
import { Alert } from '@/components/Alert';

export default function AppError({ error, reset }: { error: Error; reset: () => void }) {
  return (
    <div className="stack" style={{ maxWidth: 640 }}>
      <h1>Something went wrong</h1>
      <Alert variant="danger" title="We couldn’t load this page">
        {error.message || 'An unexpected error occurred.'}
      </Alert>
      <div>
        <button type="button" className="btn" onClick={reset}>Try again</button>
      </div>
    </div>
  );
}
