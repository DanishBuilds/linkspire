'use server';
import { revalidatePath } from 'next/cache';
import { setWorkspaceId } from '@/lib/session';

// Switch the active workspace. The id is stored in an httpOnly cookie and sent
// as X-Workspace-Id on every subsequent API call; the API validates membership.
export async function switchWorkspaceAction(formData: FormData): Promise<void> {
  const id = String(formData.get('workspace_id') || '').trim();
  if (id) await setWorkspaceId(id);
  revalidatePath('/', 'layout');
}
