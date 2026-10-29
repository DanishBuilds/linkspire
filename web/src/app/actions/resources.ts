'use server';
import { revalidatePath } from 'next/cache';
import { z } from 'zod';
import * as api from '@/lib/api';
import { ApiError } from '@/lib/api';

export interface ResState { error?: string; success?: string; secret?: string; kind?: string }

/* ---------- domains ---------- */
export async function createDomainAction(_prev: ResState, formData: FormData): Promise<ResState> {
  const hostname = String(formData.get('hostname') || '').trim().toLowerCase();
  const apex = formData.get('apex') === 'on';
  if (!hostname || /[\s/:]/.test(hostname)) return { error: 'Enter a valid hostname, e.g. go.example.com' };
  try {
    await api.createDomain(hostname, apex);
    revalidatePath('/domains');
    return { success: `Added ${hostname}. Add the DNS records below, then verification runs automatically.` };
  } catch (e) {
    return { error: e instanceof ApiError ? (e.detail || e.title) : 'Could not add the domain.' };
  }
}

/* ---------- api keys ---------- */
const keySchema = z.object({ name: z.string().trim().min(1, 'Give the key a name.'), scopes: z.string().optional() });
export async function createKeyAction(_prev: ResState, formData: FormData): Promise<ResState> {
  const parsed = keySchema.safeParse({ name: formData.get('name'), scopes: formData.get('scopes') });
  if (!parsed.success) return { error: parsed.error.issues[0]?.message ?? 'Invalid input.' };
  const scopes = (parsed.data.scopes ?? '').split(',').map((s) => s.trim()).filter(Boolean);
  try {
    const res = await api.createKey(parsed.data.name, scopes);
    revalidatePath('/api-keys');
    return { success: `Key “${parsed.data.name}” created. Copy it now — it will not be shown again.`, secret: res.key, kind: 'apikey' };
  } catch (e) {
    return { error: e instanceof ApiError ? (e.detail || e.title) : 'Could not create the key.' };
  }
}
export async function revokeKeyAction(formData: FormData): Promise<void> {
  await api.revokeKey(Number(formData.get('id')));
  revalidatePath('/api-keys');
}

/* ---------- webhooks ---------- */
export async function createWebhookAction(_prev: ResState, formData: FormData): Promise<ResState> {
  const url = String(formData.get('url') || '').trim();
  const events = String(formData.get('events') || '').split(',').map((s) => s.trim()).filter(Boolean);
  if (!url.startsWith('https://')) return { error: 'Webhook URL must start with https://' };
  if (events.length === 0) return { error: 'Select at least one event.' };
  try {
    const res = await api.createWebhook(url, events);
    revalidatePath('/webhooks');
    return { success: 'Webhook registered. Store the signing secret below — it is shown once.', secret: res.secretHex, kind: 'webhook' };
  } catch (e) {
    return { error: e instanceof ApiError ? (e.detail || e.title) : 'Could not register the webhook.' };
  }
}
export async function deleteWebhookAction(formData: FormData): Promise<void> {
  await api.deleteWebhook(Number(formData.get('id')));
  revalidatePath('/webhooks');
}
