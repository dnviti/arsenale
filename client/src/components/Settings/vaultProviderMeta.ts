import type { ExternalVaultAuthMethod, ExternalVaultType } from '../../api/externalVault.api';

interface AuthMethodMeta {
  value: ExternalVaultAuthMethod;
  label: string;
}

export interface ProviderMeta {
  value: ExternalVaultType;
  label: string;
  authMethods: AuthMethodMeta[];
  defaultMount: string;
  serverUrlPlaceholder: string;
  serverUrlLabel: string;
  secretPathHelp: string;
}

export const VAULT_PROVIDER_META: ProviderMeta[] = [
  {
    value: 'HASHICORP_VAULT',
    label: 'HashiCorp Vault',
    authMethods: [
      { value: 'TOKEN', label: 'Static token' },
      { value: 'APPROLE', label: 'AppRole' },
    ],
    defaultMount: 'secret',
    serverUrlPlaceholder: 'https://vault.example.com:8200',
    serverUrlLabel: 'Server URL',
    secretPathHelp: 'Path within the KV v2 mount, e.g. "servers/web1"',
  },
  {
    value: 'AWS_SECRETS_MANAGER',
    label: 'AWS Secrets Manager',
    authMethods: [
      { value: 'IAM_ACCESS_KEY', label: 'IAM access key' },
      { value: 'IAM_ROLE', label: 'IAM role' },
    ],
    defaultMount: '',
    serverUrlPlaceholder: 'https://secretsmanager.us-east-1.amazonaws.com',
    serverUrlLabel: 'Endpoint URL',
    secretPathHelp: 'Secret name or ARN. Append #AWSPREVIOUS for the previous version.',
  },
  {
    value: 'AZURE_KEY_VAULT',
    label: 'Azure Key Vault',
    authMethods: [
      { value: 'CLIENT_CREDENTIALS', label: 'Service principal' },
      { value: 'MANAGED_IDENTITY', label: 'Managed identity' },
    ],
    defaultMount: '',
    serverUrlPlaceholder: 'https://myvault.vault.azure.net',
    serverUrlLabel: 'Vault URI',
    secretPathHelp: 'Secret name, optionally with a version: "my-secret" or "my-secret/version-id"',
  },
  {
    value: 'GCP_SECRET_MANAGER',
    label: 'GCP Secret Manager',
    authMethods: [
      { value: 'SERVICE_ACCOUNT_KEY', label: 'Service account key' },
      { value: 'WORKLOAD_IDENTITY', label: 'Workload identity' },
    ],
    defaultMount: '',
    serverUrlPlaceholder: 'https://secretmanager.googleapis.com',
    serverUrlLabel: 'Server URL',
    secretPathHelp: 'Secret name, e.g. "my-secret" or "my-secret/versions/5"',
  },
  {
    value: 'CYBERARK_CONJUR',
    label: 'CyberArk Conjur',
    authMethods: [
      { value: 'CONJUR_API_KEY', label: 'API key' },
      { value: 'CONJUR_AUTHN_K8S', label: 'Kubernetes auth' },
    ],
    defaultMount: '',
    serverUrlPlaceholder: 'https://conjur.example.com',
    serverUrlLabel: 'Conjur URL',
    secretPathHelp: 'Variable ID with policy path, e.g. "myapp/db/password"',
  },
];

export const AUTH_METHODS_NO_CREDENTIALS_REQUIRED = new Set<ExternalVaultAuthMethod>([
  'IAM_ROLE',
  'MANAGED_IDENTITY',
  'WORKLOAD_IDENTITY',
]);

export function getProviderMeta(type: ExternalVaultType): ProviderMeta {
  return VAULT_PROVIDER_META.find((provider) => provider.value === type) ?? VAULT_PROVIDER_META[0];
}

export function providerLabel(type: ExternalVaultType): string {
  return getProviderMeta(type).label;
}

export function authMethodLabel(type: ExternalVaultType, authMethod: ExternalVaultAuthMethod): string {
  const provider = getProviderMeta(type);
  return provider.authMethods.find((method) => method.value === authMethod)?.label ?? authMethod;
}

export function requiresVaultProviderCredentials(authMethod: ExternalVaultAuthMethod) {
  return !AUTH_METHODS_NO_CREDENTIALS_REQUIRED.has(authMethod);
}
