import { useEffect, useMemo, useState } from 'react';
import {
  Eye,
  EyeOff,
  Loader2,
  LockKeyhole,
  RefreshCw,
  RotateCcw,
  Save,
} from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import type { SettingValue } from '../../api/systemSettings.api';
import { updateSystemSetting } from '../../api/systemSettings.api';
import { extractApiError } from '../../utils/apiError';
import { SettingsButtonRow, SettingsStatusBadge } from './settings-ui';

interface Props {
  setting: SettingValue;
  onUpdated: (key: string, value: unknown) => void;
}

const EMPTY_SELECT_VALUE = '__EMPTY_OPTION__';

function normalizeStringArray(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value.map((entry) => String(entry).trim()).filter(Boolean);
  }

  return String(value ?? '')
    .split(',')
    .map((entry) => entry.trim())
    .filter(Boolean);
}

function getComparableValue(setting: SettingValue, value: unknown) {
  if (setting.type === 'boolean') {
    return String(Boolean(value));
  }

  if (setting.type === 'string[]') {
    return normalizeStringArray(value).join(',');
  }

  return String(value ?? '');
}

function toApiValue(setting: SettingValue, value: unknown) {
  if (setting.type === 'boolean') {
    return Boolean(value);
  }

  if (setting.type === 'number') {
    return Number(value);
  }

  if (setting.type === 'string[]') {
    return normalizeStringArray(value).join(',');
  }

  return String(value ?? '');
}

function FieldBadges({ setting }: { setting: SettingValue }) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      {setting.source === 'env' && (
        <SettingsStatusBadge tone="warning">
          <LockKeyhole className="mr-1 size-3.5" />
          ENV Locked
        </SettingsStatusBadge>
      )}
      {setting.source === 'db' && (
        <SettingsStatusBadge tone="success">Custom</SettingsStatusBadge>
      )}
      {setting.source === 'default' && (
        <SettingsStatusBadge tone="neutral">Default</SettingsStatusBadge>
      )}
      {setting.restartRequired && (
        <SettingsStatusBadge tone="neutral">
          <RefreshCw className="mr-1 size-3.5" />
          Restart Required
        </SettingsStatusBadge>
      )}
    </div>
  );
}

export default function SettingField({ setting, onUpdated }: Props) {
  const [localValue, setLocalValue] = useState<unknown>(setting.value);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [showSensitiveValue, setShowSensitiveValue] = useState(false);

  useEffect(() => {
    setLocalValue(setting.value);
  }, [setting.key, setting.value]);

  const disabled = !setting.canEdit || setting.envLocked || saving;
  const dirty = getComparableValue(setting, localValue) !== getComparableValue(setting, setting.value);
  const arrayValue = useMemo(() => normalizeStringArray(localValue), [localValue]);
  const numberValueIsInvalid =
    setting.type === 'number'
    && String(localValue ?? '').trim() !== ''
    && Number.isNaN(Number(localValue));

  const handleSave = async (nextValue: unknown = localValue) => {
    setSaving(true);
    setError('');

    try {
      const payload = toApiValue(setting, nextValue);
      await updateSystemSetting(setting.key, payload);

      const persistedValue = setting.sensitive ? '[REDACTED]' : payload;
      setLocalValue(persistedValue);
      onUpdated(setting.key, persistedValue);
      setShowSensitiveValue(false);
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to update setting'));
    } finally {
      setSaving(false);
    }
  };

  const fieldShellClassName = 'space-y-3 rounded-xl border border-border/70 bg-background/70 p-4';

  if (setting.type === 'boolean') {
    return (
      <div className={fieldShellClassName}>
        <div className="flex items-start justify-between gap-4">
          <div className="space-y-2">
            <div className="space-y-2">
              <div className="text-sm font-medium text-foreground">{setting.label}</div>
              <FieldBadges setting={setting} />
            </div>
            <p className="text-sm leading-6 text-muted-foreground">{setting.description}</p>
          </div>
          <div className="flex items-center gap-3 pt-0.5">
            {saving && <Loader2 className="size-4 animate-spin text-muted-foreground" />}
            <Switch
              checked={Boolean(localValue)}
              disabled={disabled}
              aria-label={setting.label}
              onCheckedChange={async (checked) => {
                setLocalValue(checked);
                await handleSave(checked);
              }}
            />
          </div>
        </div>
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}
      </div>
    );
  }

  if (setting.type === 'select' && setting.options) {
    return (
      <div className={fieldShellClassName}>
        <div className="space-y-2">
          <div className="text-sm font-medium text-foreground">{setting.label}</div>
          <FieldBadges setting={setting} />
          <p className="text-sm leading-6 text-muted-foreground">{setting.description}</p>
        </div>

        <div className="flex items-center gap-3">
          <Select
            value={String(localValue ?? '') || EMPTY_SELECT_VALUE}
            onValueChange={async (value) => {
              const nextValue = value === EMPTY_SELECT_VALUE ? '' : value;
              setLocalValue(nextValue);
              await handleSave(nextValue);
            }}
            disabled={disabled}
          >
            <SelectTrigger aria-label={setting.label} className="max-w-sm">
              <SelectValue placeholder="Select a value" />
            </SelectTrigger>
            <SelectContent>
              {setting.options.map((option) => (
                <SelectItem
                  key={`${setting.key}-${option || 'empty'}`}
                  value={option || EMPTY_SELECT_VALUE}
                >
                  {option || '(disabled)'}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {saving && <Loader2 className="size-4 animate-spin text-muted-foreground" />}
        </div>

        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}
      </div>
    );
  }

  if (setting.type === 'string[]' && setting.options) {
    return (
      <div className={fieldShellClassName}>
        <div className="space-y-2">
          <div className="text-sm font-medium text-foreground">{setting.label}</div>
          <FieldBadges setting={setting} />
          <p className="text-sm leading-6 text-muted-foreground">{setting.description}</p>
        </div>

        <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
          {setting.options.map((option) => {
            const optionId = `${setting.key}-${option}`;
            const checked = arrayValue.includes(option);

            return (
              <label
                key={option}
                htmlFor={optionId}
                className="flex cursor-pointer items-start gap-3 rounded-lg border border-border/70 bg-card/50 px-3 py-3 text-sm transition-colors hover:bg-accent/40"
              >
                <Checkbox
                  id={optionId}
                  checked={checked}
                  disabled={disabled}
                  onCheckedChange={(nextChecked) => {
                    const nextValue = nextChecked
                      ? [...arrayValue, option]
                      : arrayValue.filter((entry) => entry !== option);
                    setLocalValue(nextValue);
                  }}
                />
                <span className="font-medium text-foreground">{option}</span>
              </label>
            );
          })}
        </div>

        <SettingsButtonRow>
          <Button
            type="button"
            size="sm"
            onClick={() => handleSave(arrayValue)}
            disabled={disabled || !dirty}
          >
            {saving ? <Loader2 className="animate-spin" /> : <Save />}
            Save Selection
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => setLocalValue(normalizeStringArray(setting.value))}
            disabled={saving || !dirty}
          >
            <RotateCcw />
            Reset
          </Button>
        </SettingsButtonRow>

        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}
      </div>
    );
  }

  return (
    <div className={fieldShellClassName}>
      <div className="space-y-2">
        <div className="text-sm font-medium text-foreground">{setting.label}</div>
        <FieldBadges setting={setting} />
        <p className="text-sm leading-6 text-muted-foreground">{setting.description}</p>
      </div>

      <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
        <Input
          type={setting.sensitive && !showSensitiveValue ? 'password' : setting.type === 'number' ? 'number' : 'text'}
          aria-label={setting.label}
          value={String(localValue ?? '')}
          disabled={disabled}
          className="max-w-xl"
          onChange={(event) => setLocalValue(event.target.value)}
        />

        <SettingsButtonRow>
          {setting.sensitive && !setting.envLocked && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setShowSensitiveValue((current) => !current)}
            >
              {showSensitiveValue ? <EyeOff /> : <Eye />}
              {showSensitiveValue ? 'Hide' : 'Show'}
            </Button>
          )}
          <Button
            type="button"
            size="sm"
            onClick={() => handleSave(localValue)}
            disabled={disabled || !dirty || numberValueIsInvalid}
          >
            {saving ? <Loader2 className="animate-spin" /> : <Save />}
            Save
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => setLocalValue(setting.value)}
            disabled={saving || !dirty}
          >
            <RotateCcw />
            Reset
          </Button>
        </SettingsButtonRow>
      </div>

      {numberValueIsInvalid && (
        <Alert variant="destructive">
          <AlertDescription>Enter a valid number before saving.</AlertDescription>
        </Alert>
      )}

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}
    </div>
  );
}
