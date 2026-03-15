import api from './client';
import { useAuthStore } from '../store/authStore';

export interface OAuthProviders {
  google?: boolean;
  microsoft?: boolean;
  github?: boolean;
  oidc?: boolean;
  saml?: boolean;
  ldap?: boolean;
}

export interface LinkedAccount {
  id: string;
  provider: string;
  providerEmail: string | null;
  createdAt: string;
}

export async function getOAuthProviders(): Promise<OAuthProviders> {
  const { data } = await api.get('/auth/oauth/providers');
  return data;
}

export async function getLinkedAccounts(): Promise<LinkedAccount[]> {
  const { data } = await api.get('/auth/oauth/accounts');
  return data;
}

export async function unlinkOAuthAccount(provider: string): Promise<void> {
  await api.delete(`/auth/oauth/link/${provider.toLowerCase()}`);
}

export async function setupVaultPassword(vaultPassword: string): Promise<void> {
  await api.post('/auth/oauth/vault-setup', { vaultPassword });
}

export function initiateOAuthLogin(provider: string): void {
  window.location.href = `/api/auth/oauth/${provider.toLowerCase()}`;
}

export function initiateOAuthLink(provider: string): void {
  const token = useAuthStore.getState().accessToken;
  window.location.href = `/api/auth/oauth/link/${provider.toLowerCase()}?token=${encodeURIComponent(token || '')}`;
}

export function initiateSamlLogin(): void {
  window.location.href = '/api/auth/saml';
}

export function initiateSamlLink(): void {
  const token = useAuthStore.getState().accessToken;
  window.location.href = `/api/auth/saml/link?token=${encodeURIComponent(token || '')}`;
}
