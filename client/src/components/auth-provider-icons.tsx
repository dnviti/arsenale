import { Github } from 'lucide-react';

export type AuthProviderKey =
  | 'GOOGLE'
  | 'MICROSOFT'
  | 'GITHUB'
  | 'OIDC'
  | 'SAML';

type AuthProviderServerKey = 'google' | 'microsoft' | 'github' | 'oidc' | 'saml';

export const AUTH_PROVIDER_ORDER = [
  'GOOGLE',
  'MICROSOFT',
  'GITHUB',
  'OIDC',
  'SAML',
] as const satisfies readonly AuthProviderKey[];

export const AUTH_PROVIDER_SERVER_KEYS: Record<AuthProviderKey, AuthProviderServerKey> = {
  GOOGLE: 'google',
  MICROSOFT: 'microsoft',
  GITHUB: 'github',
  OIDC: 'oidc',
  SAML: 'saml',
};

export const AUTH_PROVIDER_LABELS: Record<AuthProviderKey, string> = {
  GOOGLE: 'Google',
  MICROSOFT: 'Microsoft',
  GITHUB: 'GitHub',
  OIDC: 'SSO',
  SAML: 'SAML SSO',
};

export function getEnabledAuthProviders(
  providers: Partial<Record<AuthProviderServerKey, boolean | undefined>>,
): AuthProviderKey[] {
  return AUTH_PROVIDER_ORDER.filter((provider) =>
    Boolean(providers[AUTH_PROVIDER_SERVER_KEYS[provider]]),
  );
}

export function isAuthProviderKey(value: string): value is AuthProviderKey {
  return Object.prototype.hasOwnProperty.call(AUTH_PROVIDER_LABELS, value);
}

function iconProps(viewBox: string, fill?: string) {
  return {
    width: 20,
    height: 20,
    viewBox,
    fill,
    xmlns: 'http://www.w3.org/2000/svg',
  };
}

function GoogleIcon({ className }: { className?: string }) {
  return (
    <svg className={className} {...iconProps('0 0 48 48')}>
      <path fill="#FFC107" d="M43.6 20.5H42V20H24v8h11.3C33.6 32.7 29.2 36 24 36c-6.6 0-12-5.4-12-12s5.4-12 12-12c3 0 5.7 1.1 7.8 2.9l5.7-5.7C34.1 6.1 29.3 4 24 4 12.9 4 4 12.9 4 24s8.9 20 20 20 20-8.9 20-20c0-1.3-.1-2.4-.4-3.5z" />
      <path fill="#FF3D00" d="M6.3 14.7l6.6 4.9C14.7 15 18.9 12 24 12c3 0 5.7 1.1 7.8 2.9l5.7-5.7C34.1 6.1 29.3 4 24 4c-7.7 0-14.3 4.3-17.7 10.7z" />
      <path fill="#4CAF50" d="M24 44c5.2 0 10-2 13.6-5.2l-6.3-5.3c-2.1 1.6-4.7 2.5-7.3 2.5-5.2 0-9.6-3.3-11.3-8l-6.5 5C9.6 39.5 16.2 44 24 44z" />
      <path fill="#1976D2" d="M43.6 20.5H42V20H24v8h11.3c-.8 2.4-2.4 4.4-4.7 5.5l6.3 5.3C36.4 39.1 44 34 44 24c0-1.3-.1-2.4-.4-3.5z" />
    </svg>
  );
}

function MicrosoftIcon({ className }: { className?: string }) {
  return (
    <svg className={className} {...iconProps('0 0 21 21')}>
      <rect x="1" y="1" width="9" height="9" fill="#f25022" />
      <rect x="11" y="1" width="9" height="9" fill="#7fba00" />
      <rect x="1" y="11" width="9" height="9" fill="#00a4ef" />
      <rect x="11" y="11" width="9" height="9" fill="#ffb900" />
    </svg>
  );
}

function OidcIcon({ className }: { className?: string }) {
  return (
    <svg className={className} {...iconProps('0 0 24 24', 'currentColor')}>
      <path d="M18 8h-1V6c0-2.76-2.24-5-5-5S7 3.24 7 6v2H6c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V10c0-1.1-.9-2-2-2zM12 17c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zM15.1 8H8.9V6c0-1.71 1.39-3.1 3.1-3.1s3.1 1.39 3.1 3.1v2z" />
    </svg>
  );
}

function SamlIcon({ className }: { className?: string }) {
  return (
    <svg className={className} {...iconProps('0 0 24 24', 'currentColor')}>
      <path d="M12 1L3 5v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V5l-9-4zm0 10.99h7c-.53 4.12-3.28 7.79-7 8.94V12H5V6.3l7-3.11v8.8z" />
    </svg>
  );
}

export function AuthProviderIcon({
  provider,
  className,
}: {
  provider: AuthProviderKey;
  className?: string;
}) {
  switch (provider) {
    case 'GOOGLE':
      return <GoogleIcon className={className} />;
    case 'MICROSOFT':
      return <MicrosoftIcon className={className} />;
    case 'GITHUB':
      return <Github className={className ?? 'size-4'} />;
    case 'OIDC':
      return <OidcIcon className={className} />;
    case 'SAML':
      return <SamlIcon className={className} />;
  }
}
