import { LockKeyhole } from 'lucide-react';
import { Checkbox } from '@/components/ui/checkbox';
import { cn } from '@/lib/utils';

interface UseOverrideableSettingsOptions<T extends object> {
  value: Partial<T>;
  onChange: (updated: Partial<T>) => void;
  defaults: T;
  mode: 'global' | 'connection';
  enforcedFields?: Partial<T>;
}

export function useOverrideableSettings<T extends object>({
  value,
  onChange,
  defaults,
  mode,
  enforcedFields,
}: UseOverrideableSettingsOptions<T>) {
  const hasOwnValue = (key: keyof T) => Object.prototype.hasOwnProperty.call(value, key);

  const getValue = <K extends keyof T>(key: K): T[K] =>
    (hasOwnValue(key) ? value[key] : defaults[key]) as T[K];

  const isOverridden = (key: keyof T) =>
    mode === 'connection' && hasOwnValue(key);

  const isEnforced = (key: keyof T) =>
    enforcedFields !== undefined && Object.prototype.hasOwnProperty.call(enforcedFields, key);

  const isDisabled = (key: keyof T) =>
    isEnforced(key) || (mode === 'connection' && !isOverridden(key));

  const setField = <K extends keyof T>(key: K, nextValue: T[K] | undefined) => {
    onChange({ ...value, [key]: nextValue });
  };

  const clearField = (key: keyof T) => {
    const nextValue = { ...value };
    Reflect.deleteProperty(nextValue, key);
    onChange(nextValue);
  };

  const toggleOverride = <K extends keyof T>(key: K, ...fallbackValue: [T[K]] | []) => {
    if (isOverridden(key)) {
      clearField(key);
      return;
    }
    setField(key, fallbackValue.length > 0 ? fallbackValue[0] : defaults[key]);
  };

  return {
    getValue,
    isOverridden,
    isEnforced,
    isDisabled,
    setField,
    clearField,
    toggleOverride,
  };
}

export function SettingsOverrideToggle({
  checked,
  enforced,
  onCheckedChange,
  className,
}: {
  checked: boolean;
  enforced?: boolean;
  onCheckedChange: (checked: boolean) => void;
  className?: string;
}) {
  return (
    <label
      className={cn(
        'inline-flex items-center gap-2 rounded-full border border-border/70 bg-background px-3 py-1.5 text-xs font-medium text-foreground',
        enforced && 'border-chart-5/40 bg-chart-5/10',
        className,
      )}
    >
      <Checkbox
        checked={checked}
        disabled={enforced}
        onCheckedChange={(next) => onCheckedChange(next === true)}
        aria-label="Override this setting"
      />
      <span>Override</span>
      {enforced && (
        <span className="inline-flex items-center gap-1 text-chart-5">
          <LockKeyhole className="size-3" />
          Enforced
        </span>
      )}
    </label>
  );
}
