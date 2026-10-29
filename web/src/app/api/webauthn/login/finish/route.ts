import { NextResponse } from 'next/server';
import * as api from '@/lib/api';
import { ApiError } from '@/lib/api';
import { setSession } from '@/lib/session';

export async function POST(req: Request) {
  const body = await req.json();
  try {
    const { token } = await api.waLoginFinish(body);
    if (!token) return NextResponse.json({ error: 'no token' }, { status: 401 });
    await setSession(token);
    return NextResponse.json({ ok: true });
  } catch (e) {
    const s = e instanceof ApiError ? e.status || 500 : 500;
    return NextResponse.json({ error: e instanceof ApiError ? e.detail || e.title : 'failed' }, { status: s });
  }
}
