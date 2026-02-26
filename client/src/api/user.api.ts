import api from './client';
import type { SshTerminalConfig } from '../constants/terminalThemes';

export interface UserProfile {
  id: string;
  email: string;
  username: string | null;
  avatarData: string | null;
  sshDefaults: Partial<SshTerminalConfig> | null;
  createdAt: string;
}

export async function getProfile(): Promise<UserProfile> {
  const res = await api.get('/user/profile');
  return res.data;
}

export async function updateProfile(data: { username?: string; email?: string }): Promise<UserProfile> {
  const res = await api.put('/user/profile', data);
  return res.data;
}

export async function changePassword(oldPassword: string, newPassword: string): Promise<{ success: boolean }> {
  const res = await api.put('/user/password', { oldPassword, newPassword });
  return res.data;
}

export async function updateSshDefaults(
  data: Partial<SshTerminalConfig>
): Promise<{ id: string; sshDefaults: Partial<SshTerminalConfig> }> {
  const res = await api.put('/user/ssh-defaults', data);
  return res.data;
}

export async function uploadAvatar(avatarData: string): Promise<{ id: string; avatarData: string }> {
  const res = await api.post('/user/avatar', { avatarData });
  return res.data;
}
