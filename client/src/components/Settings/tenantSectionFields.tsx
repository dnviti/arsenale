import { Loader2, Save } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { SettingsFieldCard } from './settings-ui';
import type { TenantSelectOption } from './tenantSectionOptions';

export function TenantPolicySelectField({
  description,
  disabled,
  error,
  label,
  onValueChange,
  options,
  saving = false,
  value,
}: {
  description: string;
  disabled?: boolean;
  error?: string;
  label: string;
  onValueChange: (value: string) => void;
  options: TenantSelectOption[];
  saving?: boolean;
  value: string;
}) {
  return (
    <SettingsFieldCard
      label={label}
      description={description}
      aside={saving ? <Loader2 className="size-4 animate-spin text-muted-foreground" /> : null}
    >
      <div className="grid gap-3 sm:max-w-sm">
        <Select value={value} onValueChange={onValueChange} disabled={disabled || saving}>
          <SelectTrigger aria-label={label}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {options.map((option) => (
              <SelectItem key={`${label}-${option.value}`} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {error ? (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        ) : null}
      </div>
    </SettingsFieldCard>
  );
}

export function TenantInlineSaveField({
  description,
  disabled,
  error,
  helperText,
  inputProps,
  label,
  onChange,
  onSave,
  placeholder,
  saving = false,
  saveLabel = 'Save',
  type = 'text',
  value,
}: {
  description: string;
  disabled?: boolean;
  error?: string;
  helperText?: string;
  inputProps?: React.InputHTMLAttributes<HTMLInputElement>;
  label: string;
  onChange: (value: string) => void;
  onSave: () => void;
  placeholder?: string;
  saving?: boolean;
  saveLabel?: string;
  type?: React.HTMLInputTypeAttribute;
  value: string;
}) {
  return (
    <SettingsFieldCard label={label} description={description}>
      <div className="grid gap-3">
        <div className="flex flex-col gap-3 md:flex-row md:items-end">
          <div className="min-w-0 flex-1 space-y-2">
            <Label className="sr-only" htmlFor={label}>
              {label}
            </Label>
            <Input
              id={label}
              type={type}
              value={value}
              placeholder={placeholder}
              disabled={disabled || saving}
              onChange={(event) => onChange(event.target.value)}
              {...inputProps}
            />
          </div>
          <Button type="button" variant="outline" onClick={onSave} disabled={disabled || saving}>
            {saving ? <Loader2 className="animate-spin" /> : <Save />}
            {saving ? 'Saving...' : saveLabel}
          </Button>
        </div>
        {helperText ? <p className="text-sm text-muted-foreground">{helperText}</p> : null}
        {error ? (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        ) : null}
      </div>
    </SettingsFieldCard>
  );
}
