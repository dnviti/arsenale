import { useEffect, useState } from 'react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { SettingsPanel } from './settings-ui';
import {
  getVaultAutoLock,
  setVaultAutoLock,
  type VaultAutoLockResponse,
} from '../../api/vault.api';
import { extractApiError } from '../../utils/apiError';

const OPTIONS: { label: string; value: number | null }[] = [
  { label: 'Server default', value: null },
  { label: '5 minutes', value: 5 },
  { label: '15 minutes', value: 15 },
  { label: '30 minutes', value: 30 },
  { label: '1 hour', value: 60 },
  { label: '4 hours', value: 240 },
  { label: 'Never', value: 0 },
];

export default function VaultAutoLockSection() {
  const [data, setData] = useState<VaultAutoLockResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    getVaultAutoLock()
      .then(setData)
      .catch(() => setError('Failed to load auto-lock preference'))
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <SettingsPanel
        title="Vault Auto-Lock"
        description="Choose how long the keychain stays unlocked."
      >
        <p className="text-sm text-muted-foreground">Loading auto-lock preference...</p>
      </SettingsPanel>
    );
  }

  const tenantMax = data?.tenantMaxMinutes;
  const isOptionDisabled = (value: number | null) => {
    if (tenantMax === null || tenantMax === undefined || tenantMax <= 0) return false;
    if (value === 0) return true;
    return value !== null && value > tenantMax;
  };

  const selectValue =
    data?.autoLockMinutes === null ? 'default' : String(data?.autoLockMinutes);

  return (
    <SettingsPanel
      title="Vault Auto-Lock"
      description="Choose how long the keychain stays unlocked on this device."
    >
      <div className="space-y-4">
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        <div className="flex flex-wrap items-center gap-3">
          <Select
            value={selectValue}
            disabled={saving}
            onValueChange={async (value) => {
              setError('');
              setSaving(true);
              try {
                const result = await setVaultAutoLock(
                  value === 'default' ? null : Number(value),
                );
                setData(result);
              } catch (err: unknown) {
                setError(extractApiError(err, 'Failed to update auto-lock preference'));
              } finally {
                setSaving(false);
              }
            }}
          >
            <SelectTrigger className="w-[220px]">
              <SelectValue placeholder="Select timeout" />
            </SelectTrigger>
            <SelectContent>
              {OPTIONS.map((option) => {
                const value = option.value === null ? 'default' : String(option.value);
                return (
                  <SelectItem
                    key={value}
                    value={value}
                    disabled={isOptionDisabled(option.value)}
                  >
                    {option.label}
                  </SelectItem>
                );
              })}
            </SelectContent>
          </Select>
          {saving && <span className="text-sm text-muted-foreground">Saving...</span>}
        </div>

        <p className="text-xs leading-5 text-muted-foreground">
          Effective timeout:{' '}
          {data?.effectiveMinutes === 0 ? 'Never' : `${data?.effectiveMinutes} minutes`}
          {tenantMax != null && tenantMax > 0 && (
            <>. Your organization enforces a maximum of {tenantMax} minutes.</>
          )}
        </p>
      </div>
    </SettingsPanel>
  );
}
