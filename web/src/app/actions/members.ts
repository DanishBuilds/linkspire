'use server';
import { revalidatePath } from 'next/cache';
import * as api from '@/lib/api';
import { ApiError } from '@/lib/api';

export interface MemberState { error?: string; success?: string }

export async function changeRoleAction(_prev: MemberState, formData: FormData): Promise<MemberState> {
  const workspaceId = Number(formData.get('workspace_id'));
  const userId = Number(formData.get('user_id'));
  const role = String(formData.get('role') || '');
  if (!['owner', 'admin', 'member', 'viewer'].includes(role)) return { error: 'Choose a valid role.' };
  try {
    await api.setMemberRole(workspaceId, userId, role);
    revalidatePath('/settings/members');
    return { success: `Updated role for user #${userId} to ${role}.` };
  } catch (e) {
    return { error: e instanceof ApiError ? (e.detail || e.title) : 'Could not change the role.' };
  }
}
