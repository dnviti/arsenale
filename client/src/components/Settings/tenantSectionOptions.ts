export interface TenantSelectOption {
  label: string;
  value: string;
}

export const vaultAutoLockOptions: TenantSelectOption[] = [
  { value: 'none', label: 'No enforcement' },
  { value: '5', label: '5 minutes' },
  { value: '15', label: '15 minutes' },
  { value: '30', label: '30 minutes' },
  { value: '60', label: '1 hour' },
  { value: '240', label: '4 hours' },
];

export const maxConcurrentSessionOptions: TenantSelectOption[] = [
  { value: '0', label: 'Unlimited' },
  { value: '1', label: '1 session' },
  { value: '2', label: '2 sessions' },
  { value: '3', label: '3 sessions' },
  { value: '5', label: '5 sessions' },
  { value: '10', label: '10 sessions' },
];

export const absoluteSessionTimeoutOptions: TenantSelectOption[] = [
  { value: '0', label: 'Disabled' },
  { value: '3600', label: '1 hour' },
  { value: '14400', label: '4 hours' },
  { value: '28800', label: '8 hours' },
  { value: '43200', label: '12 hours' },
  { value: '86400', label: '24 hours' },
  { value: '604800', label: '7 days' },
];

export const loginRateLimitWindowOptions: TenantSelectOption[] = [
  { value: 'default', label: 'System default' },
  { value: '60000', label: '1 minute' },
  { value: '300000', label: '5 minutes' },
  { value: '900000', label: '15 minutes' },
  { value: '1800000', label: '30 minutes' },
  { value: '3600000', label: '1 hour' },
];

export const loginAttemptOptions: TenantSelectOption[] = [
  { value: 'default', label: 'System default' },
  { value: '3', label: '3 attempts' },
  { value: '5', label: '5 attempts' },
  { value: '10', label: '10 attempts' },
  { value: '15', label: '15 attempts' },
  { value: '20', label: '20 attempts' },
];

export const accountLockoutDurationOptions: TenantSelectOption[] = [
  { value: 'default', label: 'System default' },
  { value: '300000', label: '5 minutes' },
  { value: '900000', label: '15 minutes' },
  { value: '1800000', label: '30 minutes' },
  { value: '3600000', label: '1 hour' },
  { value: '14400000', label: '4 hours' },
];

export const impossibleTravelSpeedOptions: TenantSelectOption[] = [
  { value: 'default', label: 'System default' },
  { value: '0', label: 'Disabled' },
  { value: '500', label: '500 km/h' },
  { value: '900', label: '900 km/h' },
  { value: '1500', label: '1500 km/h' },
];

export const accessTokenExpiryOptions: TenantSelectOption[] = [
  { value: 'default', label: 'System default' },
  { value: '300', label: '5 minutes' },
  { value: '900', label: '15 minutes' },
  { value: '1800', label: '30 minutes' },
  { value: '3600', label: '1 hour' },
];

export const refreshTokenExpiryOptions: TenantSelectOption[] = [
  { value: 'default', label: 'System default' },
  { value: '86400', label: '1 day' },
  { value: '259200', label: '3 days' },
  { value: '604800', label: '7 days' },
  { value: '1209600', label: '14 days' },
  { value: '2592000', label: '30 days' },
];

export const vaultDefaultTtlOptions: TenantSelectOption[] = [
  { value: 'default', label: 'System default' },
  { value: '0', label: 'Never (0)' },
  { value: '5', label: '5 minutes' },
  { value: '15', label: '15 minutes' },
  { value: '30', label: '30 minutes' },
  { value: '60', label: '1 hour' },
  { value: '240', label: '4 hours' },
];
