import api from './client';

export async function setupSmsPhone(phoneNumber: string) {
  const { data } = await api.post('/user/2fa/sms/setup-phone', { phoneNumber });
  return data as { message: string };
}

export async function verifySmsPhone(code: string) {
  const { data } = await api.post('/user/2fa/sms/verify-phone', { code });
  return data as { verified: boolean };
}

export async function enableSmsMfa() {
  const { data } = await api.post('/user/2fa/sms/enable');
  return data as { enabled: boolean };
}

export async function sendSmsMfaDisableCode() {
  const { data } = await api.post('/user/2fa/sms/send-disable-code');
  return data as { message: string };
}

export async function disableSmsMfa(code: string) {
  const { data } = await api.post('/user/2fa/sms/disable', { code });
  return data as { enabled: boolean };
}

export async function getSmsMfaStatus() {
  const { data } = await api.get('/user/2fa/sms/status');
  return data as {
    enabled: boolean;
    phoneNumber: string | null;
    phoneVerified: boolean;
  };
}
