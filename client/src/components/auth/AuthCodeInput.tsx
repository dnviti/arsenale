import { useId, type ComponentProps, type ReactNode } from 'react';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { cn } from '@/lib/utils';

interface AuthCodeInputProps
  extends Omit<ComponentProps<typeof Input>, 'onChange' | 'value'> {
  description?: ReactNode;
  label: string;
  maxLength?: number;
  onChange: (value: string) => void;
  value: string;
}

export default function AuthCodeInput({
  className,
  description,
  id,
  label,
  maxLength = 6,
  onChange,
  placeholder = '000000',
  value,
  ...props
}: AuthCodeInputProps) {
  const generatedId = useId();
  const inputId = id ?? generatedId;

  return (
    <div className="space-y-2">
      <Label htmlFor={inputId}>{label}</Label>
      <Input
        {...props}
        id={inputId}
        autoComplete="one-time-code"
        className={cn('font-mono tracking-[0.3em]', className)}
        inputMode="numeric"
        maxLength={maxLength}
        placeholder={placeholder}
        type="text"
        value={value}
        onChange={(event) =>
          onChange(event.target.value.replace(/\D/g, '').slice(0, maxLength))}
      />
      {description ? (
        <p className="text-xs leading-5 text-muted-foreground">{description}</p>
      ) : null}
    </div>
  );
}
