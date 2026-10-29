'use server';
import { revalidatePath } from 'next/cache';
import * as api from '@/lib/api';

// Resolve an abuse report (staff only; the API enforces is_staff).
export async function resolveAbuseAction(formData: FormData): Promise<void> {
  const id = Number(formData.get('id'));
  const status = String(formData.get('status') || '');
  if (id && status) await api.resolveAbuse(id, status);
  revalidatePath('/admin/abuse');
}
