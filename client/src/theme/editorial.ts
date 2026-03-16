import { createTheme, type Theme } from '@mui/material';
import { sharedComponents, sharedShape } from './base';

const fonts = {
  sans: "'Outfit', system-ui, sans-serif",
  serif: "'Instrument Serif', Georgia, serif",
  mono: "'JetBrains Mono', 'Fira Code', monospace",
};

export const dark: Theme = createTheme({
  shape: sharedShape,
  components: sharedComponents,
  palette: {
    mode: 'dark',
    primary: { main: '#00e5a0' },
    secondary: { main: '#00cc8e' },
    background: {
      default: '#08080a',
      paper: '#0f0f12',
    },
    text: {
      primary: '#f4f4f5',
      secondary: '#a1a1aa',
      disabled: '#52525b',
    },
    divider: '#232328',
    success: { main: '#00e5a0' },
    error: { main: '#ef4444' },
    warning: { main: '#f59e0b' },
    info: { main: '#00e5a0' },
  },
  typography: {
    fontFamily: fonts.sans,
    h1: { fontFamily: fonts.serif, fontWeight: 400, letterSpacing: '-0.02em' },
    h2: { fontFamily: fonts.serif, fontWeight: 400, letterSpacing: '-0.02em' },
    h3: { fontFamily: fonts.serif, fontWeight: 400, letterSpacing: '-0.01em' },
    h4: { fontFamily: fonts.serif, fontWeight: 400, letterSpacing: '-0.01em' },
    h5: { fontFamily: fonts.serif, fontWeight: 400 },
    h6: { fontFamily: fonts.serif, fontWeight: 400 },
    subtitle1: { fontWeight: 600 },
    subtitle2: { fontWeight: 600 },
    body1: { fontWeight: 400, lineHeight: 1.7 },
    body2: { fontWeight: 400, lineHeight: 1.6 },
    caption: { fontWeight: 500, letterSpacing: '0.04em' },
    overline: { fontWeight: 500, letterSpacing: '0.15em', fontSize: '0.6875rem' },
    button: { fontWeight: 600 },
  },
});

export const light: Theme = createTheme({
  shape: sharedShape,
  components: sharedComponents,
  palette: {
    mode: 'light',
    primary: { main: '#059669' },
    secondary: { main: '#047857' },
    background: {
      default: '#f8fafb',
      paper: '#ffffff',
    },
    text: {
      primary: '#18181b',
      secondary: '#52525b',
      disabled: '#a1a1aa',
    },
    divider: '#e4e4e7',
    success: { main: '#059669' },
    error: { main: '#dc2626' },
    warning: { main: '#d97706' },
    info: { main: '#059669' },
  },
  typography: {
    fontFamily: fonts.sans,
    h1: { fontFamily: fonts.serif, fontWeight: 400, letterSpacing: '-0.02em' },
    h2: { fontFamily: fonts.serif, fontWeight: 400, letterSpacing: '-0.02em' },
    h3: { fontFamily: fonts.serif, fontWeight: 400, letterSpacing: '-0.01em' },
    h4: { fontFamily: fonts.serif, fontWeight: 400, letterSpacing: '-0.01em' },
    h5: { fontFamily: fonts.serif, fontWeight: 400 },
    h6: { fontFamily: fonts.serif, fontWeight: 400 },
    subtitle1: { fontWeight: 600 },
    subtitle2: { fontWeight: 600 },
    body1: { fontWeight: 400, lineHeight: 1.7 },
    body2: { fontWeight: 400, lineHeight: 1.6 },
    caption: { fontWeight: 500, letterSpacing: '0.04em' },
    overline: { fontWeight: 500, letterSpacing: '0.15em', fontSize: '0.6875rem' },
    button: { fontWeight: 600 },
  },
});
