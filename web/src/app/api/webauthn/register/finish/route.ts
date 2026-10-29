import { NextResponse } from 'next/server';
import * as api from '@/lib/api';
import { ApiError } from '@/lib/api';

export async function POST(req: Request) {
  const label = new URL(req.url).searchParams.get('label') || 'Passkey';
  const body = await req.json();
  try {
    return NextResponse.json(await api.waRegisterFinish(body, label));
  } catch (e) {
    const s = e instanceof ApiError ? e.status || 500 : 500;
    return NextResponse.json({ error: e instanceof ApiError ? e.detail || e.title : 'failed' }, { status: s });
  }
}
