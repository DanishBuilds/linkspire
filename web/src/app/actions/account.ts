'use server';
import { revalidatePath } from 'next/cache';
import QRCode from 'qrcode';
import * as api from '@/lib/api';
import { ApiError } from '@/lib/api';

export interface AccountState { error?: string; success?: string }
export interface MfaSetupState { error?: string; secret?: string; otpauthUrl?: string; qrDataUrl?: string }

export async function startMfaSetupAction(_prev: MfaSetupState, _formData: FormData): Promise<MfaSetupState> {
  try {
    const setup = await api.totpSetup();
    let qrDataUrl = '';
    try { qrDataUrl = await QRCode.toDataURL(setup.otpauthUrl, { margin: 1, width: 220 }); } catch { qrDataUrl = ''; }
    return { secret: setup.secret, otpauthUrl: setup.otpauthUrl, qrDataUrl };
  } catch (e) {
    return { error: e instanceof ApiError ? (e.detail || e.title) : 'Could not start two-factor setup.' };
  }
}

/* ---------- MFA ---------- */
export async function enableMfaAction(_prev: AccountState, formData: FormData): Promise<AccountState> {
  const code = String(formData.get('code') || '').trim();
  if (!/^\d{6}$/.test(code)) return { error: 'Enter the 6-digit code from your authenticator app.' };
  try {
    await api.totpEnable(code);
    revalidatePath('/settings/security');
    return { success: 'Two-factor authentication is now enabled.' };
  } catch (e) {
    return { error: e instanceof ApiError ? (e.detail || e.title) : 'Could not enable two-factor authentication.' };
  }
}

/* ---------- privacy / DSAR ---------- */
export async function createDsarAction(_prev: AccountState, formData: FormData): Promise<AccountState> {
  const type = String(formData.get('type') || '');
  const regime = String(formData.get('regime') || 'gdpr');
  const allowed = ['access', 'rectification', 'erasure', 'restriction', 'objection', 'do_not_sell'];
  if (!allowed.includes(type)) return { error: 'Choose a request type.' };
  try {
    const res = await api.createDsar(type, regime);
    revalidatePath('/settings/privacy');
    return { success: `Request received. We will respond within ${res.slaDays} days.` };
  } catch (e) {
    return { error: e instanceof ApiError ? (e.detail || e.title) : 'Could not file the request.' };
  }
}

export async function doNotSellAction(_prev: AccountState, _formData: FormData): Promise<AccountState> {
  try {
    await api.doNotSell();
    revalidatePath('/settings/privacy');
    return { success: 'Your Do Not Sell or Share preference has been recorded.' };
  } catch (e) {
    return { error: e instanceof ApiError ? (e.detail || e.title) : 'Could not record your preference.' };
  }
}
