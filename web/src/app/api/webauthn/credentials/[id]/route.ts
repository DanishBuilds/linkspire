import { NextResponse } from 'next/server';
import * as api from '@/lib/api';
import { ApiError } from '@/lib/api';

export async function DELETE(_req: Request, { params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  try {
    await api.waDelete(Number(id));
    return new NextResponse(null, { status: 204 });
  } catch (e) {
    const s = e instanceof ApiError ? e.status || 500 : 500;
    return NextResponse.json({ error: 'failed' }, { status: s });
  }
}
