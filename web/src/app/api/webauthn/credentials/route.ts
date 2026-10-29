import { NextResponse } from 'next/server';
import * as api from '@/lib/api';
import { ApiError } from '@/lib/api';

export async function GET() {
  try {
    return NextResponse.json({ data: await api.waList() });
  } catch (e) {
    const s = e instanceof ApiError ? e.status || 500 : 500;
    return NextResponse.json({ error: 'failed' }, { status: s });
  }
}
