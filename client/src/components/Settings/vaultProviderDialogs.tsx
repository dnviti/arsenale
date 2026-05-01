import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import type {
  ExternalVaultAuthMethod,
  ExternalVaultType,
} from '../../api/externalVault.api';
import { SettingsFieldCard, SettingsFieldGroup } from './settings-ui';
import { VaultProviderAuthFields } from './vaultProviderFields';
import { VAULT_PROVIDER_META, getProviderMeta } from './vaultProviderMeta';

export interface VaultProviderFormState {
  authMethod: ExternalVaultAuthMethod;
  authValues: Record<string, string>;
  cacheTtlSeconds: string;
  caCertificate: string;
  mountPath: string;
  name: string;
  namespace: string;
  providerType: ExternalVaultType;
  serverUrl: string;
}

export function createVaultProviderForm(providerType: ExternalVaultType = 'HASHICORP_VAULT'): VaultProviderFormState {
  const meta = getProviderMeta(providerType);
  return {
    authMethod: meta.authMethods[0].value,
    authValues: {},
    cacheTtlSeconds: '300',
    caCertificate: '',
    mountPath: meta.defaultMount,
    name: '',
    namespace: '',
    providerType,
    serverUrl: '',
  };
}

export function VaultProviderFormDialog({
  editing,
  form,
  formError,
  open,
  saving,
  onClose,
  onSave,
  onFormChange,
  onProviderTypeChange,
}: {
  editing: boolean;
  form: VaultProviderFormState;
  formError: string;
  open: boolean;
  saving: boolean;
  onClose: () => void;
  onSave: () => void;
  onFormChange: (nextForm: VaultProviderFormState) => void;
  onProviderTypeChange: (providerType: ExternalVaultType) => void;
}) {
  const meta = getProviderMeta(form.providerType);

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => { if (!nextOpen) onClose(); }}>
      <DialogContent className="max-h-[92vh] overflow-y-auto sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>{editing ? 'Edit vault provider' : 'Add vault provider'}</DialogTitle>
          <DialogDescription>
            Keep provider definitions minimal. Only add credentials that the selected auth method truly needs.
          </DialogDescription>
        </DialogHeader>

        {formError && (
          <Alert variant="destructive">
            <AlertDescription>{formError}</AlertDescription>
          </Alert>
        )}

        <SettingsFieldGroup>
          <SettingsFieldCard label="Provider details" description="Name the provider clearly so secret pickers stay understandable later.">
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2 md:col-span-2">
                <Label htmlFor="vault-provider-name">Name</Label>
                <Input id="vault-provider-name" value={form.name} onChange={(event) => onFormChange({ ...form, name: event.target.value })} />
              </div>

              <div className="space-y-2">
                <Label htmlFor="vault-provider-type">Provider type</Label>
                <Select value={form.providerType} disabled={editing} onValueChange={(value) => onProviderTypeChange(value as ExternalVaultType)}>
                  <SelectTrigger id="vault-provider-type" aria-label="Provider type">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {VAULT_PROVIDER_META.map((provider) => (
                      <SelectItem key={provider.value} value={provider.value}>
                        {provider.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="vault-provider-url">{meta.serverUrlLabel}</Label>
                <Input id="vault-provider-url" value={form.serverUrl} placeholder={meta.serverUrlPlaceholder} onChange={(event) => onFormChange({ ...form, serverUrl: event.target.value })} />
              </div>

              <div className="space-y-2">
                <Label htmlFor="vault-provider-auth">Auth method</Label>
                <Select
                  value={form.authMethod}
                  onValueChange={(value) => onFormChange({ ...form, authMethod: value as ExternalVaultAuthMethod, authValues: {} })}
                >
                  <SelectTrigger id="vault-provider-auth" aria-label="Auth method">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {meta.authMethods.map((authMethod) => (
                      <SelectItem key={authMethod.value} value={authMethod.value}>
                        {authMethod.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {meta.defaultMount !== '' && (
                <div className="space-y-2">
                  <Label htmlFor="vault-provider-mount">Mount path</Label>
                  <Input id="vault-provider-mount" value={form.mountPath} onChange={(event) => onFormChange({ ...form, mountPath: event.target.value })} />
                </div>
              )}

              {form.providerType === 'HASHICORP_VAULT' && (
                <div className="space-y-2">
                  <Label htmlFor="vault-provider-namespace">Namespace</Label>
                  <Input id="vault-provider-namespace" value={form.namespace} onChange={(event) => onFormChange({ ...form, namespace: event.target.value })} />
                </div>
              )}
            </div>
          </SettingsFieldCard>

          <SettingsFieldCard label="Authentication" description="Only fill the fields required by the chosen auth method.">
            <VaultProviderAuthFields
              authMethod={form.authMethod}
              isEdit={editing}
              values={form.authValues}
              onChange={(key, value) => onFormChange({ ...form, authValues: { ...form.authValues, [key]: value } })}
            />
          </SettingsFieldCard>

          <SettingsFieldCard label="Transport & cache" description="TLS trust and cache lifetime should be explicit and easy to audit.">
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="vault-provider-cache-ttl">Cache TTL (seconds)</Label>
                <Input id="vault-provider-cache-ttl" type="number" value={form.cacheTtlSeconds} onChange={(event) => onFormChange({ ...form, cacheTtlSeconds: event.target.value })} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="vault-provider-ca">CA certificate</Label>
                <Textarea
                  id="vault-provider-ca"
                  rows={4}
                  value={form.caCertificate}
                  placeholder="Optional PEM-encoded CA certificate for custom TLS trust"
                  onChange={(event) => onFormChange({ ...form, caCertificate: event.target.value })}
                />
              </div>
            </div>
          </SettingsFieldCard>
        </SettingsFieldGroup>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose} disabled={saving}>
            Cancel
          </Button>
          <Button type="button" onClick={onSave} disabled={saving}>
            {saving ? 'Saving...' : editing ? 'Save Provider' : 'Create Provider'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function VaultProviderTestDialog({
  open,
  pathValue,
  result,
  testLoading,
  testProviderType,
  onClose,
  onPathChange,
  onTest,
}: {
  open: boolean;
  pathValue: string;
  result: { success: boolean; keys?: string[]; error?: string } | null;
  testLoading: boolean;
  testProviderType: ExternalVaultType | null;
  onClose: () => void;
  onPathChange: (value: string) => void;
  onTest: () => void;
}) {
  const providerMeta = testProviderType ? getProviderMeta(testProviderType) : null;

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => { if (!nextOpen) onClose(); }}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Test vault connection</DialogTitle>
          <DialogDescription>
            Probe a specific secret path before wiring this provider into production flows.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="vault-provider-test-path">Secret path</Label>
            <Input
              id="vault-provider-test-path"
              value={pathValue}
              placeholder={providerMeta?.secretPathHelp ?? 'Secret path'}
              onChange={(event) => onPathChange(event.target.value)}
            />
            {providerMeta && (
              <p className="text-sm text-muted-foreground">{providerMeta.secretPathHelp}</p>
            )}
          </div>

          {result && (
            <Alert variant={result.success ? 'default' : 'destructive'}>
              <AlertDescription>
                {result.success
                  ? `Connection successful. Keys found: ${result.keys?.join(', ') || 'none'}`
                  : `Connection failed: ${result.error}`}
              </AlertDescription>
            </Alert>
          )}
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            Close
          </Button>
          <Button type="button" onClick={onTest} disabled={testLoading || !pathValue.trim()}>
            {testLoading ? 'Testing...' : 'Test Provider'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function VaultProviderDeleteDialog({
  name,
  open,
  onClose,
  onDelete,
}: {
  name?: string;
  open: boolean;
  onClose: () => void;
  onDelete: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={(nextOpen) => { if (!nextOpen) onClose(); }}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Delete vault provider?</DialogTitle>
          <DialogDescription>
            Remove {name ?? 'this provider'} and stop resolving secrets through it.
          </DialogDescription>
        </DialogHeader>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button type="button" variant="destructive" onClick={onDelete}>
            Delete Provider
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
