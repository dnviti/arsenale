import type { ITerminalOptions, ITheme } from '@xterm/xterm';

// ── Types ──────────────────────────────────────────────────────────────────

export interface TerminalThemeColors {
  background: string;
  foreground: string;
  cursor: string;
  selectionBackground: string;
  black: string;
  red: string;
  green: string;
  yellow: string;
  blue: string;
  magenta: string;
  cyan: string;
  white: string;
  brightBlack: string;
  brightRed: string;
  brightGreen: string;
  brightYellow: string;
  brightBlue: string;
  brightMagenta: string;
  brightCyan: string;
  brightWhite: string;
}

export interface SshTerminalConfig {
  fontFamily?: string;
  fontSize?: number;
  lineHeight?: number;
  letterSpacing?: number;
  cursorStyle?: 'block' | 'underline' | 'bar';
  cursorBlink?: boolean;
  theme?: string;
  customColors?: Partial<TerminalThemeColors>;
  scrollback?: number;
  bellStyle?: 'none' | 'sound' | 'visual';
  syncThemeWithWebUI?: boolean;
  syncLightTheme?: string;
  syncDarkTheme?: string;
}

// ── Font families ──────────────────────────────────────────────────────────

export const FONT_FAMILIES = [
  { label: 'Menlo', value: 'Menlo, Monaco, "Courier New", monospace' },
  { label: 'Fira Code', value: '"Fira Code", monospace' },
  { label: 'JetBrains Mono', value: '"JetBrains Mono", monospace' },
  { label: 'Cascadia Code', value: '"Cascadia Code", monospace' },
  { label: 'Consolas', value: 'Consolas, monospace' },
  { label: 'System Monospace', value: 'monospace' },
] as const;

// ── Defaults ───────────────────────────────────────────────────────────────

export const TERMINAL_DEFAULTS: Required<Omit<SshTerminalConfig, 'customColors'>> & {
  customColors: TerminalThemeColors;
} = {
  fontFamily: 'Menlo, Monaco, "Courier New", monospace',
  fontSize: 14,
  lineHeight: 1.0,
  letterSpacing: 0,
  cursorStyle: 'block',
  cursorBlink: true,
  theme: 'default-dark',
  customColors: {
    background: '#1a1a2e',
    foreground: '#e0e0e0',
    cursor: '#2196f3',
    selectionBackground: '#3a3a5e',
    black: '#000000',
    red: '#e06c75',
    green: '#98c379',
    yellow: '#e5c07b',
    blue: '#61afef',
    magenta: '#c678dd',
    cyan: '#56b6c2',
    white: '#abb2bf',
    brightBlack: '#5c6370',
    brightRed: '#e06c75',
    brightGreen: '#98c379',
    brightYellow: '#e5c07b',
    brightBlue: '#61afef',
    brightMagenta: '#c678dd',
    brightCyan: '#56b6c2',
    brightWhite: '#ffffff',
  },
  scrollback: 1000,
  bellStyle: 'none',
  syncThemeWithWebUI: false,
  syncLightTheme: 'solarized-light',
  syncDarkTheme: 'default-dark',
};

// ── Theme presets ──────────────────────────────────────────────────────────

export const THEME_PRESETS: Record<string, TerminalThemeColors> = {
  'default-dark': { ...TERMINAL_DEFAULTS.customColors },

  dracula: {
    background: '#282a36',
    foreground: '#f8f8f2',
    cursor: '#f8f8f2',
    selectionBackground: '#44475a',
    black: '#21222c',
    red: '#ff5555',
    green: '#50fa7b',
    yellow: '#f1fa8c',
    blue: '#bd93f9',
    magenta: '#ff79c6',
    cyan: '#8be9fd',
    white: '#f8f8f2',
    brightBlack: '#6272a4',
    brightRed: '#ff6e6e',
    brightGreen: '#69ff94',
    brightYellow: '#ffffa5',
    brightBlue: '#d6acff',
    brightMagenta: '#ff92df',
    brightCyan: '#a4ffff',
    brightWhite: '#ffffff',
  },

  'solarized-dark': {
    background: '#002b36',
    foreground: '#839496',
    cursor: '#839496',
    selectionBackground: '#073642',
    black: '#073642',
    red: '#dc322f',
    green: '#859900',
    yellow: '#b58900',
    blue: '#268bd2',
    magenta: '#d33682',
    cyan: '#2aa198',
    white: '#eee8d5',
    brightBlack: '#586e75',
    brightRed: '#cb4b16',
    brightGreen: '#586e75',
    brightYellow: '#657b83',
    brightBlue: '#839496',
    brightMagenta: '#6c71c4',
    brightCyan: '#93a1a1',
    brightWhite: '#fdf6e3',
  },

  'solarized-light': {
    background: '#fdf6e3',
    foreground: '#657b83',
    cursor: '#586e75',
    selectionBackground: '#eee8d5',
    black: '#073642',
    red: '#dc322f',
    green: '#859900',
    yellow: '#b58900',
    blue: '#268bd2',
    magenta: '#d33682',
    cyan: '#2aa198',
    white: '#eee8d5',
    brightBlack: '#586e75',
    brightRed: '#cb4b16',
    brightGreen: '#586e75',
    brightYellow: '#657b83',
    brightBlue: '#839496',
    brightMagenta: '#6c71c4',
    brightCyan: '#93a1a1',
    brightWhite: '#fdf6e3',
  },

  monokai: {
    background: '#272822',
    foreground: '#f8f8f2',
    cursor: '#f8f8f0',
    selectionBackground: '#49483e',
    black: '#272822',
    red: '#f92672',
    green: '#a6e22e',
    yellow: '#f4bf75',
    blue: '#66d9ef',
    magenta: '#ae81ff',
    cyan: '#a1efe4',
    white: '#f8f8f2',
    brightBlack: '#75715e',
    brightRed: '#f92672',
    brightGreen: '#a6e22e',
    brightYellow: '#f4bf75',
    brightBlue: '#66d9ef',
    brightMagenta: '#ae81ff',
    brightCyan: '#a1efe4',
    brightWhite: '#f9f8f5',
  },

  nord: {
    background: '#2e3440',
    foreground: '#d8dee9',
    cursor: '#d8dee9',
    selectionBackground: '#434c5e',
    black: '#3b4252',
    red: '#bf616a',
    green: '#a3be8c',
    yellow: '#ebcb8b',
    blue: '#81a1c1',
    magenta: '#b48ead',
    cyan: '#88c0d0',
    white: '#e5e9f0',
    brightBlack: '#4c566a',
    brightRed: '#bf616a',
    brightGreen: '#a3be8c',
    brightYellow: '#ebcb8b',
    brightBlue: '#81a1c1',
    brightMagenta: '#b48ead',
    brightCyan: '#8fbcbb',
    brightWhite: '#eceff4',
  },

  'one-dark': {
    background: '#282c34',
    foreground: '#abb2bf',
    cursor: '#528bff',
    selectionBackground: '#3e4451',
    black: '#282c34',
    red: '#e06c75',
    green: '#98c379',
    yellow: '#e5c07b',
    blue: '#61afef',
    magenta: '#c678dd',
    cyan: '#56b6c2',
    white: '#abb2bf',
    brightBlack: '#5c6370',
    brightRed: '#e06c75',
    brightGreen: '#98c379',
    brightYellow: '#e5c07b',
    brightBlue: '#61afef',
    brightMagenta: '#c678dd',
    brightCyan: '#56b6c2',
    brightWhite: '#ffffff',
  },

  gruvbox: {
    background: '#282828',
    foreground: '#ebdbb2',
    cursor: '#ebdbb2',
    selectionBackground: '#3c3836',
    black: '#282828',
    red: '#cc241d',
    green: '#98971a',
    yellow: '#d79921',
    blue: '#458588',
    magenta: '#b16286',
    cyan: '#689d6a',
    white: '#a89984',
    brightBlack: '#928374',
    brightRed: '#fb4934',
    brightGreen: '#b8bb26',
    brightYellow: '#fabd2f',
    brightBlue: '#83a598',
    brightMagenta: '#d3869b',
    brightCyan: '#8ec07c',
    brightWhite: '#ebdbb2',
  },

  'github-dark': {
    background: '#0d1117',
    foreground: '#c9d1d9',
    cursor: '#c9d1d9',
    selectionBackground: '#264f78',
    black: '#484f58',
    red: '#ff7b72',
    green: '#3fb950',
    yellow: '#d29922',
    blue: '#58a6ff',
    magenta: '#bc8cff',
    cyan: '#39c5cf',
    white: '#b1bac4',
    brightBlack: '#6e7681',
    brightRed: '#ffa198',
    brightGreen: '#56d364',
    brightYellow: '#e3b341',
    brightBlue: '#79c0ff',
    brightMagenta: '#d2a8ff',
    brightCyan: '#56d4dd',
    brightWhite: '#f0f6fc',
  },
};

export const THEME_PRESET_NAMES = Object.keys(THEME_PRESETS);

// ── Merge logic ────────────────────────────────────────────────────────────

export type MergedConfig = Required<Omit<SshTerminalConfig, 'customColors'>> & {
  customColors: TerminalThemeColors;
};

export function mergeTerminalConfig(
  userDefaults?: Partial<SshTerminalConfig> | null,
  connectionOverrides?: Partial<SshTerminalConfig> | null,
): MergedConfig {
  const merged: MergedConfig = { ...TERMINAL_DEFAULTS };

  // Layer 2: user defaults
  if (userDefaults) {
    for (const key of Object.keys(userDefaults) as (keyof SshTerminalConfig)[]) {
      if (key === 'customColors') continue;
      if (userDefaults[key] !== undefined) {
        (merged as Record<string, unknown>)[key] = userDefaults[key];
      }
    }
    if (userDefaults.customColors) {
      merged.customColors = { ...merged.customColors, ...userDefaults.customColors };
    }
  }

  // Layer 3: per-connection overrides
  if (connectionOverrides) {
    for (const key of Object.keys(connectionOverrides) as (keyof SshTerminalConfig)[]) {
      if (key === 'customColors') continue;
      if (connectionOverrides[key] !== undefined) {
        (merged as Record<string, unknown>)[key] = connectionOverrides[key];
      }
    }
    if (connectionOverrides.customColors) {
      merged.customColors = { ...merged.customColors, ...connectionOverrides.customColors };
    }
  }

  return merged;
}

// ── Convert to xterm.js options ────────────────────────────────────────────

export function resolveThemeForMode(
  config: MergedConfig,
  webUiMode: 'light' | 'dark',
): string {
  if (!config.syncThemeWithWebUI) return config.theme;
  return webUiMode === 'light' ? config.syncLightTheme : config.syncDarkTheme;
}

export function toXtermOptions(
  config: MergedConfig,
  webUiMode?: 'light' | 'dark',
): ITerminalOptions {
  const effectiveTheme = webUiMode
    ? resolveThemeForMode(config, webUiMode)
    : config.theme;

  const colors: TerminalThemeColors =
    effectiveTheme === 'custom'
      ? config.customColors
      : THEME_PRESETS[effectiveTheme] ?? THEME_PRESETS['default-dark'];

  const theme: ITheme = {
    background: colors.background,
    foreground: colors.foreground,
    cursor: colors.cursor,
    selectionBackground: colors.selectionBackground,
    black: colors.black,
    red: colors.red,
    green: colors.green,
    yellow: colors.yellow,
    blue: colors.blue,
    magenta: colors.magenta,
    cyan: colors.cyan,
    white: colors.white,
    brightBlack: colors.brightBlack,
    brightRed: colors.brightRed,
    brightGreen: colors.brightGreen,
    brightYellow: colors.brightYellow,
    brightBlue: colors.brightBlue,
    brightMagenta: colors.brightMagenta,
    brightCyan: colors.brightCyan,
    brightWhite: colors.brightWhite,
  };

  return {
    fontFamily: config.fontFamily,
    fontSize: config.fontSize,
    lineHeight: config.lineHeight,
    letterSpacing: config.letterSpacing,
    cursorStyle: config.cursorStyle,
    cursorBlink: config.cursorBlink,
    scrollback: config.scrollback,
    theme,
  };
}
