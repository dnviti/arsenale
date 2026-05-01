import { useCallback, useEffect, useMemo, useState } from 'react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import {
  createVaultProvider,
  deleteVaultProvider,
  listVaultProviders,
  testVaultProvider,
  updateVaultProvider,
  type CreateVaultProviderInput,
  type ExternalVaultType,
  type UpdateVaultProviderInput,
  type VaultProviderData,
} from '../../api/externalVault.api';
import { extractApiError } from '../../utils/apiError';
import { PolicyEmptyState } from './databasePolicyUi';
import {
  SettingsLoadingState,
  SettingsPanel,
  SettingsSummaryGrid,
  SettingsSummaryItem,
} from './settings-ui';
import { VaultProviderRecordCard } from './vaultProviderFields';
import {
  createVaultProviderForm,
  type VaultProviderFormState,
  VaultProviderDeleteDialog,
  VaultProviderFormDialog,
  VaultProviderTestDialog,
} from './vaultProviderDialogs';
import { requiresVaultProviderCredentials } from './vaultProviderMeta';

interface VaultProvidersSectionProps {
  tenantId: string;
}

function buildAuthPayload(values: Record<string, string>) {
  const payload: Record<string, string> = {};
  for (const [key, value] of Object.entries(values)) {
    if (value) {
      payload[key] = value;
    }
  }
  return JSON.stringify(payload);
}

function hasAuthCredentials(values: Record<string, string>) {
  return Object.values(values).some((value) => value.length > 0);
}

export default function VaultProvidersSection({ tenantId }: VaultProvidersSectionProps) {
  const [providers, setProviders] = useState<VaultProviderData[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingProvider, setEditingProvider] = useState<VaultProviderData | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<VaultProviderData | null>(null);
  const [form, setForm] = useState<VaultProviderFormState>(() => createVaultProviderForm());
  const [formError, setFormError] = useState('');
  const [saving, setSaving] = useState(false);
  const [testDialogOpen, setTestDialogOpen] = useState(false);
  const [testProvider, setTestProvider] = useState<{ id: string; type: ExternalVaultType } | null>(null);
  const [testPath, setTestPath] = useState('');
  const [testResult, setTestResult] = useState<{ success: boolean; keys?: string[]; error?: string } | null>(null);
  const [testLoading, setTestLoading] = useState(false);

  const fetchProviders = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      setProviders(await listVaultProviders());
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to load vault providers'));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (tenantId) {
      void fetchProviders();
    }
  }, [fetchProviders, tenantId]);

  const summary = useMemo(
    () => ({
      configured: providers.length,
      enabled: providers.filter((provider) => provider.enabled).length,
      disabled: providers.filter((provider) => !provider.enabled).length,
      providerKinds: new Set(providers.map((provider) => provider.providerType)).size,
    }),
    [providers],
  );

  const openCreateDialog = () => {
    setEditingProvider(null);
    setForm(createVaultProviderForm());
    setFormError('');
    setDialogOpen(true);
  };

  const openEditDialog = (provider: VaultProviderData) => {
    setEditingProvider(provider);
    setForm({
      authMethod: provider.authMethod,
      authValues: {},
      cacheTtlSeconds: String(provider.cacheTtlSeconds),
      caCertificate: '',
      mountPath: provider.mountPath,
      name: provider.name,
      namespace: provider.namespace ?? '',
      providerType: provider.providerType,
      serverUrl: provider.serverUrl,
    });
    setFormError('');
    setDialogOpen(true);
  };

  const handleProviderTypeChange = (providerType: ExternalVaultType) => {
    const nextForm = createVaultProviderForm(providerType);
    setForm((current) => ({
      ...current,
      authMethod: nextForm.authMethod,
      authValues: {},
      mountPath: nextForm.mountPath,
      namespace: '',
      providerType,
      serverUrl: '',
    }));
    setFormError('');
  };

  const handleSave = async () => {
    if (!form.name.trim() || !form.serverUrl.trim()) {
      setFormError('Name and server URL are required');
      return;
    }

    if (
      requiresVaultProviderCredentials(form.authMethod)
      && !hasAuthCredentials(form.authValues)
      && !editingProvider
    ) {
      setFormError('Authentication credentials are required for this method');
      return;
    }

    setSaving(true);
    setFormError('');
    try {
      if (editingProvider) {
        const payload: UpdateVaultProviderInput = {
          authMethod: form.authMethod,
          cacheTtlSeconds: Number.parseInt(form.cacheTtlSeconds, 10) || 300,
          mountPath: form.mountPath,
          name: form.name.trim(),
          namespace: form.namespace.trim() || null,
          providerType: form.providerType,
          serverUrl: form.serverUrl.trim(),
          ...(form.caCertificate.trim() ? { caCertificate: form.caCertificate.trim() } : {}),
          ...(hasAuthCredentials(form.authValues) ? { authPayload: buildAuthPayload(form.authValues) } : {}),
        };
        await updateVaultProvider(editingProvider.id, payload);
      } else {
        const payload: CreateVaultProviderInput = {
          authMethod: form.authMethod,
          authPayload: buildAuthPayload(form.authValues),
          cacheTtlSeconds: Number.parseInt(form.cacheTtlSeconds, 10) || 300,
          mountPath: form.mountPath,
          name: form.name.trim(),
          providerType: form.providerType,
          serverUrl: form.serverUrl.trim(),
          ...(form.namespace.trim() ? { namespace: form.namespace.trim() } : {}),
          ...(form.caCertificate.trim() ? { caCertificate: form.caCertificate.trim() } : {}),
        };
        await createVaultProvider(payload);
      }

      setDialogOpen(false);
      await fetchProviders();
    } catch (err: unknown) {
      setFormError(extractApiError(err, 'Failed to save vault provider'));
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setError('');
    try {
      await deleteVaultProvider(deleteTarget.id);
      setDeleteTarget(null);
      await fetchProviders();
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to delete vault provider'));
    }
  };

  const handleToggleEnabled = async (provider: VaultProviderData) => {
    setError('');
    try {
      await updateVaultProvider(provider.id, { enabled: !provider.enabled });
      await fetchProviders();
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to toggle vault provider'));
    }
  };

  const openTestDialog = (provider: VaultProviderData) => {
    setTestProvider({ id: provider.id, type: provider.providerType });
    setTestPath('');
    setTestResult(null);
    setTestDialogOpen(true);
  };

  const handleTest = async () => {
    if (!testProvider) return;
    setTestLoading(true);
    setTestResult(null);
    try {
      setTestResult(await testVaultProvider(testProvider.id, testPath));
    } catch (err: unknown) {
      setTestResult({ success: false, error: extractApiError(err, 'Test failed') });
    } finally {
      setTestLoading(false);
    }
  };

  return (
    <>
      <SettingsPanel
        title="External vault providers"
        description="Reference secrets from external vault systems without forcing people through multiple disconnected setup screens."
        heading={(
          <Button type="button" onClick={openCreateDialog}>
            Add Provider
          </Button>
        )}
      >
        <div className="space-y-5">
          <SettingsSummaryGrid className="xl:grid-cols-4">
            <SettingsSummaryItem label="Configured" value={summary.configured} />
            <SettingsSummaryItem label="Enabled" value={summary.enabled} />
            <SettingsSummaryItem label="Disabled" value={summary.disabled} />
            <SettingsSummaryItem label="Provider types" value={summary.providerKinds} />
          </SettingsSummaryGrid>

          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          {loading ? (
            <SettingsLoadingState message="Loading external vault providers..." />
          ) : providers.length === 0 ? (
            <PolicyEmptyState
              title="No vault providers configured"
              description="Add HashiCorp Vault, AWS Secrets Manager, Azure Key Vault, GCP Secret Manager, or CyberArk Conjur when credentials should stay outside the built-in keychain."
            />
          ) : (
            <div className="space-y-4">
              {providers.map((provider) => (
                <VaultProviderRecordCard
                  key={provider.id}
                  provider={provider}
                  onTest={() => openTestDialog(provider)}
                  onEdit={() => openEditDialog(provider)}
                  onDelete={() => setDeleteTarget(provider)}
                  onToggleEnabled={() => { void handleToggleEnabled(provider); }}
                />
              ))}
            </div>
          )}
        </div>
      </SettingsPanel>

      <VaultProviderFormDialog
        open={dialogOpen}
        editing={Boolean(editingProvider)}
        form={form}
        formError={formError}
        saving={saving}
        onClose={() => setDialogOpen(false)}
        onSave={handleSave}
        onFormChange={setForm}
        onProviderTypeChange={handleProviderTypeChange}
      />

      <VaultProviderTestDialog
        open={testDialogOpen}
        pathValue={testPath}
        result={testResult}
        testLoading={testLoading}
        testProviderType={testProvider?.type ?? null}
        onClose={() => setTestDialogOpen(false)}
        onPathChange={setTestPath}
        onTest={handleTest}
      />

      <VaultProviderDeleteDialog
        open={Boolean(deleteTarget)}
        name={deleteTarget?.name}
        onClose={() => setDeleteTarget(null)}
        onDelete={handleDelete}
      />
    </>
  );
}
