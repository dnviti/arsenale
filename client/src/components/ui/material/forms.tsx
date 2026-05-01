import {
  cloneElement,
  createContext,
  useContext,
  type CSSProperties,
  type ChangeEvent,
  type ComponentProps,
  type HTMLAttributes,
  type InputHTMLAttributes,
  type ReactElement,
  type ReactNode,
  type TextareaHTMLAttributes,
} from 'react';
import { Circle } from 'lucide-react';
import { Checkbox as ShadCheckbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Slider as ShadSlider } from '@/components/ui/slider';
import { Switch as ShadSwitch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
import {
  ToggleGroup,
  ToggleGroupItem,
} from '@/components/ui/toggle-group';
import { cn } from '@/lib/utils';
import { useSxClassName, type SxProp } from './theme';

interface CommonProps {
  [key: string]: any;
  children?: ReactNode;
  className?: string;
  style?: CSSProperties;
  sx?: SxProp;
}

const SelectContext = createContext(false);
const RadioGroupContext = createContext<{
  name?: string;
  onChange?: (event: ChangeEvent<HTMLInputElement>, value: string) => void;
  value?: string;
}>({});

function InputAdornment({
  position,
  className,
  children,
}: {
  children?: ReactNode;
  className?: string;
  position: 'end' | 'start';
}) {
  return (
    <span
      className={cn(
        'inline-flex items-center text-muted-foreground',
        position === 'start' ? 'mr-2' : 'ml-2',
        className,
      )}
    >
      {children}
    </span>
  );
}

function FormControl({
  children,
  fullWidth,
  sx,
  style,
  className,
  ...props
}: CommonProps &
  HTMLAttributes<HTMLDivElement> & {
    fullWidth?: boolean;
}) {
  const sxClassName = useSxClassName(sx);
  return (
    <div
      className={cn('space-y-2', fullWidth && 'w-full', sxClassName, className)}
      style={style}
      {...props}
    >
      {children}
    </div>
  );
}

function InputLabel({
  children,
  sx,
  style,
  className,
  shrink,
  ...props
}: CommonProps &
  ComponentProps<typeof Label> & {
    shrink?: boolean;
  }) {
  const sxClassName = useSxClassName(sx);
  return (
    <Label className={cn(shrink && 'opacity-100', sxClassName, className)} style={style} {...props}>
      {children}
    </Label>
  );
}

function renderWithAdornments(
  input: ReactElement<any>,
  startAdornment?: ReactNode,
  endAdornment?: ReactNode,
) {
  if (!startAdornment && !endAdornment) {
    return input;
  }

  return (
    <div className="flex items-center rounded-lg border border-input bg-background px-3 focus-within:ring-2 focus-within:ring-ring/40">
      {startAdornment}
      {cloneElement(input, {
        className: cn('border-0 bg-transparent px-0 shadow-none focus-visible:ring-0', input.props.className),
      } as Record<string, unknown>)}
      {endAdornment}
    </div>
  );
}

function TextField({
  label,
  helperText,
  error,
  fullWidth,
  multiline,
  rows,
  size = 'medium',
  sx,
  style,
  className,
  slotProps,
  InputProps,
  inputProps,
  value,
  ...props
}: CommonProps &
  Omit<InputHTMLAttributes<HTMLInputElement>, 'size'> &
  Omit<TextareaHTMLAttributes<HTMLTextAreaElement>, 'size'> & {
    error?: boolean;
    fullWidth?: boolean;
    helperText?: ReactNode;
    InputProps?: {
      endAdornment?: ReactNode;
      readOnly?: boolean;
      startAdornment?: ReactNode;
    };
    inputProps?: InputHTMLAttributes<HTMLInputElement>;
    label?: ReactNode;
    multiline?: boolean;
    rows?: number;
    size?: 'small' | 'medium';
    slotProps?: {
      htmlInput?: InputHTMLAttributes<HTMLInputElement>;
      input?: InputHTMLAttributes<HTMLInputElement> & { readOnly?: boolean; startAdornment?: ReactNode; endAdornment?: ReactNode; sx?: CSSProperties };
      inputLabel?: { shrink?: boolean };
    };
  }) {
  const sxClassName = useSxClassName(sx);
  const inputStyle = slotProps?.input?.sx;
  const inputClassName = cn(
    fullWidth && 'w-full',
    size === 'small' && 'h-9 text-sm',
  );
  const adornmentStart = InputProps?.startAdornment ?? slotProps?.input?.startAdornment;
  const adornmentEnd = InputProps?.endAdornment ?? slotProps?.input?.endAdornment;

  const sharedProps = {
    ...props,
    ...inputProps,
    ...slotProps?.htmlInput,
    value,
    readOnly: InputProps?.readOnly ?? slotProps?.input?.readOnly,
  };

  const field = multiline ? (
    <Textarea
      rows={rows}
      className={cn(inputClassName, error && 'border-destructive')}
      style={inputStyle}
      {...(sharedProps as TextareaHTMLAttributes<HTMLTextAreaElement>)}
    />
  ) : (
    <Input
      className={cn(inputClassName, error && 'border-destructive')}
      style={inputStyle}
      {...sharedProps}
    />
  );

  return (
    <div className={cn('space-y-2', fullWidth && 'w-full', sxClassName, className)} style={style}>
      {label ? <InputLabel shrink={slotProps?.inputLabel?.shrink}>{label}</InputLabel> : null}
      {renderWithAdornments(field, adornmentStart, adornmentEnd)}
      {helperText ? (
        <div className={cn('text-xs', error ? 'text-destructive' : 'text-muted-foreground')}>
          {helperText}
        </div>
      ) : null}
    </div>
  );
}

function Select({
  children,
  fullWidth,
  size: muiSize = 'medium',
  sx,
  style,
  className,
  onChange,
  ...props
}: {
  children?: ReactNode;
  className?: string;
  fullWidth?: boolean;
  onChange?: (event: any) => void;
  size?: any;
  style?: CSSProperties;
  sx?: SxProp;
  [key: string]: any;
}) {
  const sxClassName = useSxClassName(sx);
  return (
    <SelectContext.Provider value>
      <select
        className={cn(
          'flex h-10 rounded-lg border border-input bg-background px-3 py-2 text-sm shadow-xs outline-none focus:ring-2 focus:ring-ring/40 disabled:cursor-not-allowed disabled:opacity-50',
          fullWidth && 'w-full',
          muiSize === 'small' && 'h-9',
          sxClassName,
          className,
        )}
        style={style}
        onChange={onChange}
        {...props}
      >
        {children}
      </select>
    </SelectContext.Provider>
  );
}

function MenuItem({
  children,
  className,
  value,
  sx,
  style,
  ...props
}: CommonProps &
  HTMLAttributes<HTMLButtonElement> & {
    value?: string | number;
  }) {
  const insideSelect = useContext(SelectContext);
  const sxClassName = useSxClassName(sx);

  if (insideSelect) {
    return (
      <option value={value} className={cn(sxClassName, className)} style={style}>
        {children}
      </option>
    );
  }

  return (
    <button
      type="button"
      className={cn('flex w-full items-center rounded-lg px-2 py-1.5 text-left text-sm hover:bg-accent', sxClassName, className)}
      style={style}
      {...props}
    >
      {children}
    </button>
  );
}

function Switch({
  checked,
  onChange,
  sx,
  style,
  className,
  ...props
}: {
  checked?: boolean;
  className?: string;
  onChange?: (event: any, checked: any) => void;
  style?: CSSProperties;
  sx?: SxProp;
  [key: string]: any;
}) {
  const sxClassName = useSxClassName(sx);
  return (
    <ShadSwitch
      checked={checked}
      className={cn(sxClassName, className)}
      style={style}
      onCheckedChange={(next) => onChange?.({} as unknown, next)}
      {...props}
    />
  );
}

function Checkbox({
  checked,
  onChange,
  sx,
  style,
  className,
  ...props
}: {
  checked?: boolean;
  className?: string;
  onChange?: (event: any, checked: any) => void;
  style?: CSSProperties;
  sx?: SxProp;
  [key: string]: any;
}) {
  const sxClassName = useSxClassName(sx);
  return (
    <ShadCheckbox
      checked={checked}
      className={cn(sxClassName, className)}
      style={style}
      onCheckedChange={(next) => {
        const checkedValue = Boolean(next);
        onChange?.({ target: { checked: checkedValue } } as ChangeEvent<HTMLInputElement>, checkedValue);
      }}
      {...props}
    />
  );
}

function Radio({
  value,
  checked,
  onChange,
  sx,
  style,
  className,
  ...props
}: CommonProps &
  InputHTMLAttributes<HTMLInputElement>) {
  const group = useContext(RadioGroupContext);
  const sxClassName = useSxClassName(sx);
  const isChecked = checked ?? group.value === value;

  return (
    <label className={cn('inline-flex cursor-pointer items-center justify-center', sxClassName, className)} style={style}>
      <input
        type="radio"
        className="sr-only"
        name={group.name}
        checked={Boolean(isChecked)}
        value={value}
        onChange={(event) => {
          group.onChange?.(event, event.target.value);
          onChange?.(event);
        }}
        {...props}
      />
      <span className={cn('flex size-4 items-center justify-center rounded-full border', isChecked && 'border-primary')}>
        {isChecked ? <Circle className="size-2.5 fill-primary text-primary" /> : null}
      </span>
    </label>
  );
}

function RadioGroup({
  children,
  name,
  value,
  onChange,
  sx,
  style,
  className,
}: CommonProps & {
  name?: string;
  onChange?: (event: ChangeEvent<HTMLInputElement>, value: string) => void;
  value?: string;
}) {
  const sxClassName = useSxClassName(sx);
  return (
    <RadioGroupContext.Provider value={{ name, onChange, value }}>
      <div className={cn('flex flex-col gap-2', sxClassName, className)} style={style}>
        {children}
      </div>
    </RadioGroupContext.Provider>
  );
}

function FormControlLabel({
  control,
  label,
  sx,
  style,
  className,
}: CommonProps & {
  control: ReactNode;
  label: ReactNode;
}) {
  const sxClassName = useSxClassName(sx);
  return (
    <label className={cn('flex items-center gap-3 text-sm', sxClassName, className)} style={style}>
      {control}
      <span>{label}</span>
    </label>
  );
}

function Slider({
  value,
  defaultValue,
  onChange,
  sx,
  style,
  className,
  ...props
}: {
  className?: string;
  defaultValue?: any;
  onChange?: (event: any, value: any) => void;
  style?: CSSProperties;
  sx?: SxProp;
  value?: any;
  [key: string]: any;
}) {
  const sxClassName = useSxClassName(sx);
  const resolvedValue = Array.isArray(value) ? value : value == null ? undefined : [value];
  const resolvedDefault = Array.isArray(defaultValue)
    ? defaultValue
    : defaultValue == null
      ? undefined
      : [defaultValue];

  return (
    <ShadSlider
      className={cn(sxClassName, className)}
      style={style}
      value={resolvedValue}
      defaultValue={resolvedDefault}
      onValueChange={(next) => onChange?.({}, next.length === 1 ? next[0] : next)}
      {...props}
    />
  );
}

function ToggleButton({
  value,
  children,
  className,
  ...props
}: {
  children?: ReactNode;
  className?: string;
  value?: any;
  [key: string]: any;
}) {
  return (
    <ToggleGroupItem value={value} className={className} {...props}>
      {children}
    </ToggleGroupItem>
  );
}

function ToggleButtonGroup({
  value,
  exclusive,
  onChange,
  className,
  children,
  ...props
}: {
  children?: ReactNode;
  className?: string;
  exclusive?: boolean;
  onChange?: (event: any, value: any) => void;
  value?: any;
  [key: string]: any;
}) {
  return (
    <ToggleGroup
      type={exclusive ? 'single' : 'multiple'}
      value={value == null ? undefined : value}
      className={className}
      onValueChange={(next: any) => onChange?.({}, next || null)}
      {...props}
    >
      {children}
    </ToggleGroup>
  );
}

function Autocomplete({
  options = [],
  value,
  inputValue,
  onChange,
  onInputChange,
  renderInput,
  getOptionLabel,
  renderOption,
  renderTags,
  multiple,
  loading,
  noOptionsText = 'No options',
}: {
  filterOptions?: (options: any[], state: any) => any[];
  getOptionLabel?: (option: any) => string;
  inputValue?: string;
  isOptionEqualToValue?: (option: any, value: any) => boolean;
  loading?: boolean;
  multiple?: boolean;
  noOptionsText?: ReactNode;
  onChange?: (event: any, value: any, reason?: any) => void;
  onInputChange?: (event: any, value: any, reason?: any) => void;
  options?: any[];
  renderInput?: (params: any) => ReactNode;
  renderOption?: (props: any, option: any, state: any, ownerState: any) => ReactNode;
  renderTags?: (value: any[], getTagProps: (args: any) => any) => ReactNode;
  value?: any;
  [key: string]: any;
}) {
  const resolvedInputValue =
    inputValue
    ?? (multiple
      ? ''
      : value
        ? (getOptionLabel?.(value) ?? value.label ?? String(value))
        : '');

  const params = {
    InputProps: {
      startAdornment: null,
      endAdornment: loading ? null : null,
    },
    inputProps: {
      value: resolvedInputValue,
      onChange: (event: any) => onInputChange?.(event, event.target.value, 'input'),
    },
  };

  return (
    <div className="space-y-2">
      {renderInput?.(params)}
      {Array.isArray(value) && renderTags ? renderTags(value, ({ index }: { index: number }) => ({ key: index })) : null}
      <div className="max-h-64 overflow-auto rounded-xl border">
        {options.length === 0 ? (
          <div className="px-3 py-2 text-sm text-muted-foreground">{noOptionsText}</div>
        ) : (
          options.map((option: any, index: number) => {
            const label = getOptionLabel?.(option) ?? option.label ?? String(option);
            const optionProps = {
              key: option.id ?? option.value ?? index,
              onClick: (event: any) => onChange?.(event, multiple ? [...(value ?? []), option] : option, 'selectOption'),
            };
            return renderOption
              ? renderOption(optionProps, option, { selected: false }, {})
              : (
                <button
                  key={optionProps.key}
                  type="button"
                  className="flex w-full items-center px-3 py-2 text-left text-sm hover:bg-accent"
                  onClick={optionProps.onClick}
                >
                  {label}
                </button>
              );
          })
        )}
      </div>
    </div>
  );
}

export {
  Checkbox,
  FormControl,
  FormControlLabel,
  InputAdornment,
  InputLabel,
  MenuItem,
  Radio,
  RadioGroup,
  Select,
  Slider,
  Switch,
  TextField,
  Autocomplete,
  ToggleButton,
  ToggleButtonGroup,
};
