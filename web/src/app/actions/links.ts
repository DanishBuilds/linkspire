'use server';
import { revalidatePath } from 'next/cache';
import { redirect } from 'next/navigation';
import { z } from 'zod';
import * as api from '@/lib/api';
import { ApiError } from '@/lib/api';

export interface LinkFormState { error?: string; success?: string; shortUrl?: string; code?: string; domainId?: number }

function toRfc3339(local: string | null): string | undefined {
  if (!local) return undefined;
  // datetime-local gives "YYYY-MM-DDTHH:mm" with no zone; interpret as UTC.
  if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(local)) return `${local}:00Z`;
  return local;
}

const createSchema = z.object({
  destination: z.string().url('Destination must be a valid http(s) URL.'),
  alias: z.string().trim().optional(),
  domain_id: z.coerce.number().int().optional(),
  redirect_type: z.coerce.number().int().optional(),
  password: z.string().optional(),
  click_limit: z.coerce.number().int().positive().optional(),
  title: z.string().trim().optional(),
});

export async function createLinkAction(_prev: LinkFormState, formData: FormData): Promise<LinkFormState> {
  const parsed = createSchema.safeParse({
    destination: formData.get('destination'),
    alias: formData.get('alias') || undefined,
    domain_id: formData.get('domain_id') || undefined,
    redirect_type: formData.get('redirect_type') || undefined,
    password: formData.get('password') || undefined,
    click_limit: formData.get('click_limit') || undefined,
    title: formData.get('title') || undefined,
  });
  if (!parsed.success) return { error: parsed.error.issues[0]?.message ?? 'Invalid input.' };

  const expires = toRfc3339((formData.get('expires_at') as string) || null);
  try {
    const link = await api.createLink({
      destination: parsed.data.destination,
      alias: parsed.data.alias,
      domain_id: parsed.data.domain_id && parsed.data.domain_id > 0 ? parsed.data.domain_id : undefined,
      redirect_type: parsed.data.redirect_type,
      password: parsed.data.password,
      click_limit: parsed.data.click_limit,
      title: parsed.data.title,
      expires_at: expires,
    });
    revalidatePath('/links');
    revalidatePath('/dashboard');
    return { success: 'Short link created.', shortUrl: link.shortUrl, code: link.code, domainId: link.domainId };
  } catch (e) {
    return { error: e instanceof ApiError ? (e.detail || e.title) : 'Could not create the link.' };
  }
}

export async function patchLinkAction(_prev: LinkFormState, formData: FormData): Promise<LinkFormState> {
  const domainId = Number(formData.get('domain_id'));
  const code = String(formData.get('code'));
  const destination = String(formData.get('destination') || '');
  const status = String(formData.get('status') || '');
  const redirect_type = Number(formData.get('redirect_type') || 0);
  try {
    await api.patchLink(domainId, code, {
      destination: destination || undefined,
      status: status || undefined,
      redirect_type: redirect_type || undefined,
    });
    revalidatePath(`/links/${domainId}/${code}`);
    revalidatePath('/links');
    return { success: 'Link updated.' };
  } catch (e) {
    return { error: e instanceof ApiError ? (e.detail || e.title) : 'Could not update the link.' };
  }
}

export async function deleteLinkAction(formData: FormData): Promise<void> {
  const domainId = Number(formData.get('domain_id'));
  const code = String(formData.get('code'));
  await api.deleteLink(domainId, code);
  revalidatePath('/links');
  revalidatePath('/dashboard');
  redirect('/links');
}

export async function requestQrAction(_prev: LinkFormState, formData: FormData): Promise<LinkFormState> {
  const domainId = Number(formData.get('domain_id'));
  const code = String(formData.get('code'));
  const format = (String(formData.get('format') || 'png') === 'svg' ? 'svg' : 'png') as 'png' | 'svg';
  try {
    await api.requestQR(domainId, code, { format });
    return { success: `QR generation queued (${format.toUpperCase()}). The worker will render it shortly.` };
  } catch (e) {
    return { error: e instanceof ApiError ? (e.detail || e.title) : 'Could not queue the QR code.' };
  }
}

export interface BulkState { error?: string; created?: number; failed?: number; errors?: string[] }

export async function bulkCreateAction(_prev: BulkState, formData: FormData): Promise<BulkState> {
  const raw = String(formData.get('urls') || '');
  const urls = raw.split(/\r?\n/).map((u) => u.trim()).filter(Boolean);
  if (urls.length === 0) return { error: 'Paste at least one URL, one per line.' };
  if (urls.length > 100) return { error: 'Bulk create accepts at most 100 URLs at a time.' };
  try {
    const res = await api.bulkCreate(urls.map((destination) => ({ destination })));
    revalidatePath('/links');
    revalidatePath('/dashboard');
    return { created: res.ok.length, failed: res.errors.length, errors: res.errors.slice(0, 10) };
  } catch (e) {
    return { error: e instanceof ApiError ? (e.detail || e.title) : 'Bulk create failed.' };
  }
}

