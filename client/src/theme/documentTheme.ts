import { alpha, mixColors } from '@/lib/color';
import type { AppTheme, ThemeMode, ThemeName } from './index';

interface ThemeFonts {
  body: string;
  heading: string;
  mono: string;
}

const themeFonts: Record<ThemeName, ThemeFonts> = {
  editorial: {
    body: "'Outfit', system-ui, sans-serif",
    heading: "'Instrument Serif', Georgia, serif",
    mono: "'JetBrains Mono', 'Fira Code', monospace",
  },
  primer: {
    body: "'Figtree', system-ui, sans-serif",
    heading: "'Figtree', system-ui, sans-serif",
    mono: "'Source Code Pro', 'Fira Code', monospace",
  },
  tanuki: {
    body: "'Urbanist', system-ui, sans-serif",
    heading: "'Manrope', system-ui, sans-serif",
    mono: "'Fira Code', 'JetBrains Mono', monospace",
  },
  monokai: {
    body: "'Rubik', system-ui, sans-serif",
    heading: "'Syne', system-ui, sans-serif",
    mono: "'Space Mono', 'Fira Code', monospace",
  },
  solarized: {
    body: "'Libre Franklin', system-ui, sans-serif",
    heading: "'Newsreader', Georgia, serif",
    mono: "'Inconsolata', 'JetBrains Mono', monospace",
  },
  onedark: {
    body: "'Karla', system-ui, sans-serif",
    heading: "'Bricolage Grotesque', system-ui, sans-serif",
    mono: "'JetBrains Mono', 'Fira Code', monospace",
  },
};

export function applyDocumentTheme(theme: AppTheme, themeName: ThemeName, mode: ThemeMode) {
  const root = document.documentElement;
  const fonts = themeFonts[themeName];
  const paper = theme.palette.background.paper;
  const background = theme.palette.background.default;
  const divider = theme.palette.divider;
  const primary = theme.palette.primary.main;
  const secondary = theme.palette.secondary.main;
  const error = theme.palette.error.main;
  const success = theme.palette.success.main;
  const warning = theme.palette.warning.main;
  const info = theme.palette.info.main;
  const secondarySurfaceBlend = mode === 'dark' ? 0.22 : 0.1;
  const accentSurfaceBlend = mode === 'dark' ? 0.18 : 0.08;
  const mutedSurfaceBlend = mode === 'dark' ? 0.38 : 0.2;

  const variables = {
    '--background': background,
    '--foreground': theme.palette.text.primary,
    '--card': paper,
    '--card-foreground': theme.palette.text.primary,
    '--popover': paper,
    '--popover-foreground': theme.palette.text.primary,
    '--primary': primary,
    '--primary-foreground': theme.palette.getContrastText(primary),
    '--secondary': mixColors(secondary, paper, secondarySurfaceBlend),
    '--secondary-foreground': theme.palette.text.primary,
    '--muted': mixColors(divider, paper, mutedSurfaceBlend),
    '--muted-foreground': theme.palette.text.secondary,
    '--accent': mixColors(primary, paper, accentSurfaceBlend),
    '--accent-foreground': theme.palette.text.primary,
    '--destructive': error,
    '--destructive-foreground': theme.palette.getContrastText(error),
    '--border': divider,
    '--input': divider,
    '--ring': primary,
    '--chart-1': primary,
    '--chart-2': secondary,
    '--chart-3': info,
    '--chart-4': success,
    '--chart-5': warning,
    '--sidebar': paper,
    '--sidebar-foreground': theme.palette.text.primary,
    '--sidebar-primary': primary,
    '--sidebar-primary-foreground': theme.palette.getContrastText(primary),
    '--sidebar-accent': mixColors(primary, paper, accentSurfaceBlend),
    '--sidebar-accent-foreground': theme.palette.text.primary,
    '--sidebar-border': divider,
    '--sidebar-ring': primary,
    '--radius': `${theme.shape.borderRadius}px`,
    '--font-sans': fonts.body,
    '--font-heading': fonts.heading,
    '--font-mono': fonts.mono,
    '--arsenale-accent': primary,
    '--arsenale-bg': background,
    '--arsenale-border': divider,
    '--arsenale-muted': theme.palette.text.disabled,
    '--arsenale-primary-08': alpha(primary, 0.08),
    '--arsenale-primary-10': alpha(primary, 0.1),
    '--arsenale-primary-12': alpha(primary, 0.12),
    '--arsenale-primary-14': alpha(primary, 0.14),
    '--arsenale-primary-20': alpha(primary, 0.2),
    '--arsenale-primary-26': alpha(primary, 0.26),
    '--arsenale-primary-33': alpha(primary, 0.33),
    '--arsenale-primary-40': alpha(primary, 0.4),
    '--arsenale-primary-65': alpha(primary, 0.65),
    '--arsenale-secondary-12': alpha(secondary, 0.12),
    '--arsenale-error-14': alpha(error, 0.14),
    '--arsenale-error-26': alpha(error, 0.26),
    '--arsenale-warning-14': alpha(warning, 0.14),
    '--arsenale-warning-26': alpha(warning, 0.26),
    '--arsenale-info-14': alpha(info, 0.14),
    '--arsenale-info-26': alpha(info, 0.26),
    '--arsenale-surface-55': alpha(paper, 0.55),
    '--arsenale-surface-84': alpha(background, 0.84),
    '--arsenale-surface-90': alpha(background, 0.9),
  };

  root.dataset.themeName = themeName;
  root.dataset.themeMode = mode;
  root.classList.toggle('dark', mode === 'dark');

  for (const [name, value] of Object.entries(variables)) {
    root.style.setProperty(name, value);
  }
}
