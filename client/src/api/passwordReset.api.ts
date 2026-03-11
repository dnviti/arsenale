import api from './client';

export async function forgotPasswordApi(email: string): Promise<{ message: string }> {
  const { data } = await api.post('/auth/forgot-password', { email });
  return data;
}

export async function validateResetTokenApi(token: string): Promise<{
  valid: boolean;
  requiresSmsVerification: boolean;
  maskedPhone?: string;
  hasRecoveryKey: boolean;
}> {
  const { data } = await api.post('/auth/reset-password/validate', { token });
  return data;
}

export async function requestResetSmsCodeApi(token: string): Promise<{ message: string }> {
  const { data } = await api.post('/auth/reset-password/request-sms', { token });
  return data;
}

export async function completePasswordResetApi(params: {
  token: string;
  newPassword: string;
  smsCode?: string;
  recoveryKey?: string;
}): Promise<{ success: boolean; vaultPreserved: boolean; newRecoveryKey?: string }> {
  const { data } = await api.post('/auth/reset-password/complete', params);
  return data;
}
