import { darken, getContrastText, lighten } from '@/lib/color';

export type ThemeMode = 'light' | 'dark';
export type ThemeName = 'editorial' | 'primer' | 'tanuki' | 'monokai' | 'solarized' | 'onedark';

export interface ThemeInfo {
  name: ThemeName;
  label: string;
  description: string;
  accent: string; // dark-mode accent color for preview swatch
  accentLight: string; // light-mode accent color for preview swatch
}

export const themeRegistry: ThemeInfo[] = [
  { name: 'editorial', label: 'Dark Editorial Precision', description: 'Emerald accent, serif headings', accent: '#00e5a0', accentLight: '#059669' },
  { name: 'primer', label: 'Primer Clarity', description: 'GitHub-inspired, trusted blue', accent: '#58a6ff', accentLight: '#0969da' },
  { name: 'tanuki', label: 'Tanuki Bold', description: 'GitLab-inspired, purple + orange', accent: '#7c3aed', accentLight: '#6e49cb' },
  { name: 'monokai', label: 'Neon Syntax', description: 'Monokai-inspired, multi-color', accent: '#a6e22e', accentLight: '#6d8c14' },
  { name: 'solarized', label: 'Precision Spectrum', description: 'Solarized-inspired, cyan accent', accent: '#2aa198', accentLight: '#2aa198' },
  { name: 'onedark', label: 'Atom Equilibrium', description: 'OneDark-inspired, balanced blue', accent: '#61afef', accentLight: '#4078f2' },
];

export const THEME_NAMES: ThemeName[] = themeRegistry.map((t) => t.name);

export interface AppPaletteColor {
  main: string;
  light: string;
  dark: string;
  contrastText: string;
}

export interface AppTheme {
  shape: {
    borderRadius: number;
  };
  zIndex: {
    drawer: number;
  };
  palette: {
    mode: ThemeMode;
    primary: AppPaletteColor;
    secondary: AppPaletteColor;
    background: {
      default: string;
      paper: string;
    };
    text: {
      primary: string;
      secondary: string;
      disabled: string;
    };
    divider: string;
    success: AppPaletteColor;
    error: AppPaletteColor;
    warning: AppPaletteColor;
    info: AppPaletteColor;
    common: {
      black: string;
      white: string;
    };
    getContrastText: (color: string) => string;
  };
  typography: {
    fontFamily: string;
    h4: { fontFamily: string };
    h5: { fontFamily: string };
    h6: { fontFamily: string };
    body2: { fontSize: string };
  };
}

interface ThemeDefinition {
  mode: ThemeMode;
  primary: string;
  secondary: string;
  background: {
    default: string;
    paper: string;
  };
  text: {
    primary: string;
    secondary: string;
    disabled: string;
  };
  divider: string;
  success: string;
  error: string;
  warning: string;
  info: string;
  fonts: {
    body: string;
    heading: string;
  };
}

function createPaletteColor(main: string): AppPaletteColor {
  return {
    main,
    light: lighten(main, 0.16),
    dark: darken(main, 0.14),
    contrastText: getContrastText(main),
  };
}

function createAppTheme(definition: ThemeDefinition): AppTheme {
  return {
    shape: {
      borderRadius: 12,
    },
    zIndex: {
      drawer: 1200,
    },
    palette: {
      mode: definition.mode,
      primary: createPaletteColor(definition.primary),
      secondary: createPaletteColor(definition.secondary),
      background: definition.background,
      text: definition.text,
      divider: definition.divider,
      success: createPaletteColor(definition.success),
      error: createPaletteColor(definition.error),
      warning: createPaletteColor(definition.warning),
      info: createPaletteColor(definition.info),
      common: {
        black: '#000000',
        white: '#ffffff',
      },
      getContrastText,
    },
    typography: {
      fontFamily: definition.fonts.body,
      h4: { fontFamily: definition.fonts.heading },
      h5: { fontFamily: definition.fonts.heading },
      h6: { fontFamily: definition.fonts.heading },
      body2: { fontSize: '0.875rem' },
    },
  };
}

export const themes: Record<ThemeName, Record<ThemeMode, AppTheme>> = {
  editorial: {
    dark: createAppTheme({
      mode: 'dark',
      primary: '#00e5a0',
      secondary: '#00cc8e',
      background: { default: '#08080a', paper: '#0f0f12' },
      text: { primary: '#f4f4f5', secondary: '#a1a1aa', disabled: '#52525b' },
      divider: '#232328',
      success: '#00e5a0',
      error: '#ef4444',
      warning: '#f59e0b',
      info: '#00e5a0',
      fonts: {
        body: "'Outfit', system-ui, sans-serif",
        heading: "'Instrument Serif', Georgia, serif",
      },
    }),
    light: createAppTheme({
      mode: 'light',
      primary: '#059669',
      secondary: '#047857',
      background: { default: '#f8fafb', paper: '#ffffff' },
      text: { primary: '#18181b', secondary: '#52525b', disabled: '#a1a1aa' },
      divider: '#e4e4e7',
      success: '#059669',
      error: '#dc2626',
      warning: '#d97706',
      info: '#059669',
      fonts: {
        body: "'Outfit', system-ui, sans-serif",
        heading: "'Instrument Serif', Georgia, serif",
      },
    }),
  },
  primer: {
    dark: createAppTheme({
      mode: 'dark',
      primary: '#58a6ff',
      secondary: '#1f6feb',
      background: { default: '#0d1117', paper: '#161b22' },
      text: { primary: '#f0f6fc', secondary: '#8b949e', disabled: '#484f58' },
      divider: '#30363d',
      success: '#3fb950',
      error: '#f85149',
      warning: '#d29922',
      info: '#58a6ff',
      fonts: {
        body: "'Figtree', system-ui, sans-serif",
        heading: "'Figtree', system-ui, sans-serif",
      },
    }),
    light: createAppTheme({
      mode: 'light',
      primary: '#0969da',
      secondary: '#0550ae',
      background: { default: '#ffffff', paper: '#f6f8fa' },
      text: { primary: '#1f2328', secondary: '#656d76', disabled: '#6e7781' },
      divider: '#d0d7de',
      success: '#1a7f37',
      error: '#cf222e',
      warning: '#9a6700',
      info: '#0969da',
      fonts: {
        body: "'Figtree', system-ui, sans-serif",
        heading: "'Figtree', system-ui, sans-serif",
      },
    }),
  },
  tanuki: {
    dark: createAppTheme({
      mode: 'dark',
      primary: '#7c3aed',
      secondary: '#fc6d26',
      background: { default: '#0e0e1a', paper: '#14142b' },
      text: { primary: '#ededf0', secondary: '#9e9eb8', disabled: '#5a5a7a' },
      divider: '#2d2d52',
      success: '#2da160',
      error: '#ec5941',
      warning: '#fc6d26',
      info: '#a78bfa',
      fonts: {
        body: "'Urbanist', system-ui, sans-serif",
        heading: "'Manrope', system-ui, sans-serif",
      },
    }),
    light: createAppTheme({
      mode: 'light',
      primary: '#6e49cb',
      secondary: '#e24329',
      background: { default: '#fafafa', paper: '#ffffff' },
      text: { primary: '#1e1e2e', secondary: '#55556e', disabled: '#8e8ea8' },
      divider: '#dcdbe5',
      success: '#217645',
      error: '#c91c1c',
      warning: '#e24329',
      info: '#8b5cf6',
      fonts: {
        body: "'Urbanist', system-ui, sans-serif",
        heading: "'Manrope', system-ui, sans-serif",
      },
    }),
  },
  monokai: {
    dark: createAppTheme({
      mode: 'dark',
      primary: '#a6e22e',
      secondary: '#f92672',
      background: { default: '#1e1f1c', paper: '#272822' },
      text: { primary: '#f8f8f2', secondary: '#b8b8a8', disabled: '#75715e' },
      divider: '#454640',
      success: '#a6e22e',
      error: '#fd971f',
      warning: '#e6db74',
      info: '#66d9ef',
      fonts: {
        body: "'Rubik', system-ui, sans-serif",
        heading: "'Syne', system-ui, sans-serif",
      },
    }),
    light: createAppTheme({
      mode: 'light',
      primary: '#6d8c14',
      secondary: '#d4145a',
      background: { default: '#fdf8ee', paper: '#faf4e4' },
      text: { primary: '#2e2f2a', secondary: '#6b6c60', disabled: '#9e9e8a' },
      divider: '#d9d3c3',
      success: '#6d8c14',
      error: '#c87010',
      warning: '#9e8e20',
      info: '#1a8a9e',
      fonts: {
        body: "'Rubik', system-ui, sans-serif",
        heading: "'Syne', system-ui, sans-serif",
      },
    }),
  },
  solarized: {
    dark: createAppTheme({
      mode: 'dark',
      primary: '#2aa198',
      secondary: '#268bd2',
      background: { default: '#002b36', paper: '#073642' },
      text: { primary: '#eee8d5', secondary: '#93a1a1', disabled: '#586e75' },
      divider: '#0d4a5a',
      success: '#859900',
      error: '#dc322f',
      warning: '#b58900',
      info: '#268bd2',
      fonts: {
        body: "'Libre Franklin', system-ui, sans-serif",
        heading: "'Newsreader', Georgia, serif",
      },
    }),
    light: createAppTheme({
      mode: 'light',
      primary: '#2aa198',
      secondary: '#268bd2',
      background: { default: '#fdf6e3', paper: '#eee8d5' },
      text: { primary: '#073642', secondary: '#586e75', disabled: '#93a1a1' },
      divider: '#d6ceb5',
      success: '#859900',
      error: '#dc322f',
      warning: '#b58900',
      info: '#268bd2',
      fonts: {
        body: "'Libre Franklin', system-ui, sans-serif",
        heading: "'Newsreader', Georgia, serif",
      },
    }),
  },
  onedark: {
    dark: createAppTheme({
      mode: 'dark',
      primary: '#61afef',
      secondary: '#528bcc',
      background: { default: '#282c34', paper: '#21252b' },
      text: { primary: '#abb2bf', secondary: '#636d83', disabled: '#5c6370' },
      divider: '#3e4451',
      success: '#98c379',
      error: '#e06c75',
      warning: '#e5c07b',
      info: '#56b6c2',
      fonts: {
        body: "'Karla', system-ui, sans-serif",
        heading: "'Bricolage Grotesque', system-ui, sans-serif",
      },
    }),
    light: createAppTheme({
      mode: 'light',
      primary: '#4078f2',
      secondary: '#3568d4',
      background: { default: '#fafafa', paper: '#f0f0f0' },
      text: { primary: '#383a42', secondary: '#696c77', disabled: '#a0a1a7' },
      divider: '#d4d4d4',
      success: '#50a14f',
      error: '#e45649',
      warning: '#c18401',
      info: '#0184bc',
      fonts: {
        body: "'Karla', system-ui, sans-serif",
        heading: "'Bricolage Grotesque', system-ui, sans-serif",
      },
    }),
  },
};
