import { FlaskConical, PencilLine, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
import type {
  ExternalVaultAuthMethod,
  VaultProviderData,
} from '../../api/externalVault.api';
import { SettingsButtonRow, SettingsStatusBadge } from './settings-ui';
import { authMethodLabel, providerLabel } from './vaultProviderMeta';

function FieldHint({ children }: { children: string }) {
  return <p className="text-sm leading-6 text-muted-foreground">{children}</p>;
}

function TextFieldRow({
  label,
  value,
  onChange,
  required = false,
  placeholder,
  multiline = false,
  rows = 4,
  type = 'text',
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  required?: boolean;
  placeholder?: string;
  multiline?: boolean;
  rows?: number;
  type?: string;
}) {
  return (
    <div className="space-y-2">
      <Label>
        {label}
        {required ? ' *' : ''}
      </Label>
      {multiline ? (
        <Textarea
          value={value}
          rows={rows}
          placeholder={placeholder}
          onChange={(event) => onChange(event.target.value)}
        />
      ) : (
        <Input
          type={type}
          value={value}
          placeholder={placeholder}
          onChange={(event) => onChange(event.target.value)}
        />
      )}
    </div>
  );
}

export function VaultProviderAuthFields({
  authMethod,
  isEdit,
  values,
  onChange,
}: {
  authMethod: ExternalVaultAuthMethod;
  isEdit: boolean;
  values: Record<string, string>;
  onChange: (key: string, value: string) => void;
}) {
  const placeholder = isEdit ? 'Leave blank to keep the existing credential' : undefined;
  const required = !isEdit;

  switch (authMethod) {
    case 'TOKEN':
      return (
        <TextFieldRow
          label="Vault token"
          type="password"
          value={values.token ?? ''}
          required={required}
          placeholder={placeholder}
          onChange={(value) => onChange('token', value)}
        />
      );
    case 'APPROLE':
      return (
        <div className="grid gap-4 md:grid-cols-2">
          <TextFieldRow label="Role ID" value={values.roleId ?? ''} required={required} placeholder={placeholder} onChange={(value) => onChange('roleId', value)} />
          <TextFieldRow label="Secret ID" type="password" value={values.secretId ?? ''} required={required} placeholder={placeholder} onChange={(value) => onChange('secretId', value)} />
        </div>
      );
    case 'IAM_ACCESS_KEY':
      return (
        <div className="grid gap-4 md:grid-cols-2">
          <TextFieldRow label="Access key ID" value={values.accessKeyId ?? ''} required={required} placeholder={placeholder} onChange={(value) => onChange('accessKeyId', value)} />
          <TextFieldRow label="Secret access key" type="password" value={values.secretAccessKey ?? ''} required={required} placeholder={placeholder} onChange={(value) => onChange('secretAccessKey', value)} />
          <div className="md:col-span-2">
            <TextFieldRow label="Region" value={values.region ?? ''} placeholder="us-east-1 (default)" onChange={(value) => onChange('region', value)} />
          </div>
        </div>
      );
    case 'IAM_ROLE':
      return (
        <div className="space-y-4">
          <TextFieldRow label="Region" value={values.region ?? ''} placeholder="us-east-1 (default)" onChange={(value) => onChange('region', value)} />
          <TextFieldRow label="Role ARN" value={values.roleArn ?? ''} placeholder="arn:aws:iam::123456789:role/my-role" onChange={(value) => onChange('roleArn', value)} />
          <FieldHint>Credentials come from the environment, such as IRSA, an instance profile, or runtime IAM injection.</FieldHint>
        </div>
      );
    case 'CLIENT_CREDENTIALS':
      return (
        <div className="grid gap-4 md:grid-cols-2">
          <TextFieldRow label="Azure tenant ID" value={values.tenantId ?? ''} required={required} placeholder={placeholder} onChange={(value) => onChange('tenantId', value)} />
          <TextFieldRow label="Client ID" value={values.clientId ?? ''} required={required} placeholder={placeholder} onChange={(value) => onChange('clientId', value)} />
          <div className="md:col-span-2">
            <TextFieldRow label="Client secret" type="password" value={values.clientSecret ?? ''} required={required} placeholder={placeholder} onChange={(value) => onChange('clientSecret', value)} />
          </div>
        </div>
      );
    case 'MANAGED_IDENTITY':
      return (
        <div className="space-y-4">
          <TextFieldRow label="Client ID" value={values.clientId ?? ''} placeholder="Leave blank for the system-assigned identity" onChange={(value) => onChange('clientId', value)} />
          <FieldHint>Managed identity only works when this deployment runs on Azure.</FieldHint>
        </div>
      );
    case 'SERVICE_ACCOUNT_KEY':
      return (
        <div className="space-y-4">
          <TextFieldRow label="Service account key JSON" value={values.serviceAccountKey ?? ''} required={required} placeholder={isEdit ? 'Leave blank to keep the existing key' : 'Paste the full JSON key file'} multiline rows={5} onChange={(value) => onChange('serviceAccountKey', value)} />
          <TextFieldRow label="Project ID" value={values.projectId ?? ''} placeholder="Optional if the key already contains it" onChange={(value) => onChange('projectId', value)} />
        </div>
      );
    case 'WORKLOAD_IDENTITY':
      return (
        <div className="space-y-4">
          <TextFieldRow label="Project ID" value={values.projectId ?? ''} required={required} placeholder={placeholder} onChange={(value) => onChange('projectId', value)} />
          <FieldHint>Workload identity uses the platform metadata server, so it only works from supported GCP runtimes.</FieldHint>
        </div>
      );
    case 'CONJUR_API_KEY':
      return (
        <div className="grid gap-4 md:grid-cols-2">
          <TextFieldRow label="Account" value={values.account ?? ''} required={required} placeholder={placeholder} onChange={(value) => onChange('account', value)} />
          <TextFieldRow label="Login (host ID)" value={values.login ?? ''} required={required} placeholder={placeholder} onChange={(value) => onChange('login', value)} />
          <div className="md:col-span-2">
            <TextFieldRow label="API key" type="password" value={values.apiKey ?? ''} required={required} placeholder={placeholder} onChange={(value) => onChange('apiKey', value)} />
          </div>
        </div>
      );
    case 'CONJUR_AUTHN_K8S':
      return (
        <div className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <TextFieldRow label="Account" value={values.account ?? ''} required={required} placeholder={placeholder} onChange={(value) => onChange('account', value)} />
            <TextFieldRow label="Service ID" value={values.serviceId ?? ''} required={required} placeholder={placeholder} onChange={(value) => onChange('serviceId', value)} />
          </div>
          <TextFieldRow label="Host ID" value={values.hostId ?? ''} placeholder="Optional host identity override" onChange={(value) => onChange('hostId', value)} />
          <FieldHint>Kubernetes auth uses the workload service account token instead of a user-supplied secret.</FieldHint>
        </div>
      );
    default:
      return null;
  }
}

export function VaultProviderRecordCard({
  provider,
  onDelete,
  onEdit,
  onTest,
  onToggleEnabled,
}: {
  provider: VaultProviderData;
  onDelete: () => void;
  onEdit: () => void;
  onTest: () => void;
  onToggleEnabled: () => void;
}) {
  return (
    <div className="rounded-2xl border border-border/70 bg-background/70 p-4">
      <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div className="space-y-3">
          <div className="space-y-2">
            <div className="flex flex-wrap items-center gap-2">
              <div className="text-sm font-semibold text-foreground">{provider.name}</div>
              <SettingsStatusBadge tone={provider.enabled ? 'success' : 'neutral'}>
                {provider.enabled ? 'Enabled' : 'Disabled'}
              </SettingsStatusBadge>
            </div>
            <p className="break-all text-sm leading-6 text-muted-foreground">{provider.serverUrl}</p>
          </div>

          <div className="flex flex-wrap gap-2">
            <SettingsStatusBadge tone="neutral">{providerLabel(provider.providerType)}</SettingsStatusBadge>
            <SettingsStatusBadge tone="neutral">{authMethodLabel(provider.providerType, provider.authMethod)}</SettingsStatusBadge>
            {provider.namespace && (
              <SettingsStatusBadge tone="neutral">Namespace: {provider.namespace}</SettingsStatusBadge>
            )}
            {provider.mountPath && (
              <SettingsStatusBadge tone="neutral">Mount: {provider.mountPath}</SettingsStatusBadge>
            )}
          </div>

          <div className="text-xs text-muted-foreground">
            Cache TTL {provider.cacheTtlSeconds}s · Updated {new Date(provider.updatedAt).toLocaleDateString()}
          </div>
        </div>

        <div className="space-y-3 xl:shrink-0">
          <label className="flex items-center justify-between gap-3 rounded-xl border border-border/70 bg-background px-3 py-2 text-sm text-foreground">
            <span>Enabled</span>
            <Switch checked={provider.enabled} onCheckedChange={onToggleEnabled} aria-label={`Toggle ${provider.name}`} />
          </label>

          <SettingsButtonRow className="justify-end">
            <Button type="button" variant="outline" size="sm" onClick={onTest}>
              <FlaskConical className="size-4" />
              Test
            </Button>
            <Button type="button" variant="outline" size="sm" onClick={onEdit}>
              <PencilLine className="size-4" />
              Edit
            </Button>
            <Button type="button" variant="outline" size="sm" onClick={onDelete}>
              <Trash2 className="size-4" />
              Delete
            </Button>
          </SettingsButtonRow>
        </div>
      </div>
    </div>
  );
}
