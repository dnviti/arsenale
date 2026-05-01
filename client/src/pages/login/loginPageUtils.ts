export type LoginStep =
  | 'passkey'
  | 'credentials'
  | 'mfa-choice'
  | 'email'
  | 'totp'
  | 'sms'
  | 'webauthn'
  | 'mfa-setup'
  | 'tenant-select';

export const LOGIN_STEP_SUBTITLES: Record<LoginStep, string> = {
  passkey: 'Sign in with your passkey',
  credentials: 'Sign in to manage your connections',
  'mfa-choice': 'Choose your verification method',
  email: 'Enter the 6-digit code sent to your email',
  totp: 'Enter the 6-digit code from your authenticator app',
  sms: 'Enter the 6-digit code sent to your phone',
  webauthn: 'Verify your identity with your security key or passkey',
  'mfa-setup': 'Your organization requires two-factor authentication',
  'tenant-select': 'Select the organization you want to work in',
};
