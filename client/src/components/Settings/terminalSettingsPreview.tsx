import type { ReactNode } from 'react';
import { cn } from '@/lib/utils';
import type { TerminalThemeColors } from '../../constants/terminalThemes';

export const ANSI_COLOR_KEYS: (keyof TerminalThemeColors)[] = [
  'background',
  'foreground',
  'cursor',
  'selectionBackground',
  'black',
  'red',
  'green',
  'yellow',
  'blue',
  'magenta',
  'cyan',
  'white',
  'brightBlack',
  'brightRed',
  'brightGreen',
  'brightYellow',
  'brightBlue',
  'brightMagenta',
  'brightCyan',
  'brightWhite',
];

export function themeLabel(name: string): string {
  return name
    .split('-')
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

function ColorSwatch({
  color,
  label,
}: {
  color: string;
  label: string;
}) {
  return (
    <div
      title={label}
      className="size-4 rounded-[4px] border border-black/10"
      style={{ backgroundColor: color }}
    />
  );
}

export function TerminalPalettePreview({
  colors,
  className,
}: {
  colors: TerminalThemeColors;
  className?: string;
}) {
  return (
    <div className={cn('flex flex-wrap gap-1', className)}>
      <ColorSwatch color={colors.background} label="background" />
      <ColorSwatch color={colors.foreground} label="foreground" />
      <ColorSwatch color={colors.red} label="red" />
      <ColorSwatch color={colors.green} label="green" />
      <ColorSwatch color={colors.yellow} label="yellow" />
      <ColorSwatch color={colors.blue} label="blue" />
      <ColorSwatch color={colors.magenta} label="magenta" />
      <ColorSwatch color={colors.cyan} label="cyan" />
    </div>
  );
}

export function TerminalThemeOptionCard({
  label,
  colors,
  selected,
  disabled,
  description,
  onSelect,
}: {
  label: string;
  colors?: TerminalThemeColors;
  selected: boolean;
  disabled?: boolean;
  description?: ReactNode;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onSelect}
      disabled={disabled}
      className={cn(
        'rounded-xl border p-3 text-left transition-colors',
        selected ? 'border-primary bg-primary/10 shadow-sm' : 'border-border/70 bg-background',
        disabled && 'cursor-not-allowed opacity-50',
      )}
      style={colors ? { backgroundColor: colors.background } : undefined}
    >
      <div
        className="text-xs font-semibold"
        style={colors ? { color: colors.foreground } : undefined}
      >
        {label}
      </div>
      {description && (
        <div
          className="mt-1 text-xs leading-5 text-muted-foreground"
          style={colors ? { color: colors.foreground } : undefined}
        >
          {description}
        </div>
      )}
      {colors && <TerminalPalettePreview colors={colors} className="mt-3" />}
    </button>
  );
}

export function TerminalLivePreview({
  colors,
  fontFamily,
  fontSize,
  lineHeight,
  letterSpacing,
  cursorBlink,
}: {
  colors: TerminalThemeColors;
  fontFamily: string;
  fontSize: number;
  lineHeight: number;
  letterSpacing: number;
  cursorBlink: boolean;
}) {
  return (
    <div
      className="overflow-hidden rounded-xl border p-4 shadow-inner"
      style={{
        backgroundColor: colors.background,
        fontFamily,
        fontSize: `${fontSize}px`,
        lineHeight,
        letterSpacing: `${letterSpacing}px`,
      }}
    >
      <div className="space-y-1">
        <div>
          <span style={{ color: colors.green }}>user@host</span>
          <span style={{ color: colors.foreground }}>:</span>
          <span style={{ color: colors.blue }}>~/projects</span>
          <span style={{ color: colors.foreground }}>$ ls -la</span>
        </div>
        <div style={{ color: colors.foreground }}>total 42</div>
        <div>
          <span style={{ color: colors.blue }}>drwxr-xr-x</span>
          <span style={{ color: colors.foreground }}> 5 user group 4096 </span>
          <span style={{ color: colors.cyan }}>src/</span>
        </div>
        <div>
          <span style={{ color: colors.foreground }}>-rw-r--r-- 1 user group 1234 </span>
          <span style={{ color: colors.yellow }}>README.md</span>
        </div>
        <div>
          <span style={{ color: colors.foreground }}>-rwxr-xr-x 1 user group 5678 </span>
          <span style={{ color: colors.green }}>build.sh</span>
        </div>
        <div>
          <span style={{ color: colors.red }}>error:</span>
          <span style={{ color: colors.foreground }}> something went wrong</span>
        </div>
        <div>
          <span style={{ color: colors.magenta }}>warning:</span>
          <span style={{ color: colors.foreground }}> check configuration</span>
        </div>
        <div>
          <span style={{ color: colors.green }}>user@host</span>
          <span style={{ color: colors.foreground }}>:</span>
          <span style={{ color: colors.blue }}>~/projects</span>
          <span style={{ color: colors.foreground }}>$ </span>
          <span
            className={cn('inline-block w-[0.6em] align-text-bottom', cursorBlink && 'animate-pulse')}
            style={{
              backgroundColor: colors.cursor,
              height: '1.05em',
            }}
          />
        </div>
      </div>
    </div>
  );
}
