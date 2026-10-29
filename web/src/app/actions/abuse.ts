'use server';
import { z } from 'zod';
import * as api from '@/lib/api';
import { ApiError } from '@/lib/api';

export interface AbuseState { error?: string; success?: string }

const schema = z.object({
  short_url: z.string().trim().min(1, 'Enter the short link you are reporting.'),
  reason: z.string().trim().min(10, 'Please describe the problem (at least 10 characters).'),
});

export async function reportAbuseAction(_prev: AbuseState, formData: FormData): Promise<AbuseState> {
  const parsed = schema.safeParse({ short_url: formData.get('short_url'), reason: formData.get('reason') });
  if (!parsed.success) return { error: parsed.error.issues[0]?.message ?? 'Invalid input.' };
  try {
    await api.reportAbuse(parsed.data.short_url, parsed.data.reason);
    return { success: 'Thank you. Your report has entered our review queue.' };
  } catch (e) {
    return { error: e instanceof ApiError ? (e.detail || e.title) : 'Could not submit the report. Please try again.' };
  }
}
