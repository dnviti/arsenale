import api from './client';

export async function setup2FA() {
  const { data } = await api.post('/user/2fa/setup');
  return data as { secret: string; otpauthUri: string };
}

export async function verify2FA(code: string) {
  const { data } = await api.post('/user/2fa/verify', { code });
  return data as { enabled: boolean };
}

export async function disable2FA(code: string) {
  const { data } = await api.post('/user/2fa/disable', { code });
  return data as { enabled: boolean };
}

export async function get2FAStatus() {
  const { data } = await api.get('/user/2fa/status');
  return data as { enabled: boolean };
}
