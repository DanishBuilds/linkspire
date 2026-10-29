'use server';
import { redirect } from 'next/navigation';
import { z } from 'zod';
import * as api from '@/lib/api';
import { ApiError } from '@/lib/api';
import { setSession, clearSession } from '@/lib/session';

export interface FormState { error?: string; needsTotp?: boolean; email?: string }

const loginSchema = z.object({
  email: z.string().email('Enter a valid email address.'),
  password: z.string().min(1, 'Password is required.'),
  totp: z.string().trim().optional(),
});

export async function loginAction(_prev: FormState, formData: FormData): Promise<FormState> {
  const parsed = loginSchema.safeParse({
    email: formData.get('email'),
    password: formData.get('password'),
    totp: formData.get('totp'),
  });
  if (!parsed.success) {
    return { error: parsed.error.issues[0]?.message ?? 'Invalid input.', email: String(formData.get('email') ?? '') };
  }
  const { email, password, totp } = parsed.data;
  try {
    const res = await api.login({ email, password, totp: totp || undefined });
    await setSession(res.token);
  } catch (e) {
    if (e instanceof ApiError) {
      if (e.detail.toLowerCase().includes('totp')) {
        return { error: 'Enter the 6-digit code from your authenticator app.', needsTotp: true, email };
      }
      return { error: e.status === 401 ? 'Invalid email or password.' : e.detail || e.title, email };
    }
    return { error: 'Something went wrong. Please try again.', email };
  }
  const next = String(formData.get('next') || '/dashboard');
  redirect(next.startsWith('/') ? next : '/dashboard');
}

const registerSchema = z.object({
  email: z.string().email('Enter a valid email address.'),
  password: z.string().min(10, 'Password must be at least 10 characters.'),
  workspace: z.string().trim().optional(),
  home_region: z.enum(['us', 'eu']),
});

export async function registerAction(_prev: FormState, formData: FormData): Promise<FormState> {
  const parsed = registerSchema.safeParse({
    email: formData.get('email'),
    password: formData.get('password'),
    workspace: formData.get('workspace'),
    home_region: formData.get('home_region'),
  });
  if (!parsed.success) {
    return { error: parsed.error.issues[0]?.message ?? 'Invalid input.', email: String(formData.get('email') ?? '') };
  }
  try {
    const res = await api.register({
      email: parsed.data.email,
      password: parsed.data.password,
      workspace: parsed.data.workspace || undefined,
      home_region: parsed.data.home_region,
    });
    await setSession(res.token);
  } catch (e) {
    if (e instanceof ApiError) {
      return { error: e.status === 409 ? 'That email is already registered.' : e.detail || e.title, email: parsed.data.email };
    }
    return { error: 'Something went wrong. Please try again.', email: parsed.data.email };
  }
  redirect('/dashboard');
}

export async function logoutAction(): Promise<void> {
  await clearSession();
  redirect('/login');
}
