import api from './client';

export async function authorizeCliDevice(userCode: string) {
  const { data } = await api.post('/cli/auth/device/authorize', {
    user_code: userCode,
  });
  return data as { message: string };
}
