import { createTheme, type Theme } from '@mui/material';
import { sharedComponents, sharedShape } from './base';

const fonts = {
  sans: "'Figtree', system-ui, sans-serif",
  mono: "'Source Code Pro', 'Fira Code', monospace",
};

export const dark: Theme = createTheme({
  shape: sharedShape,
  components: sharedComponents,
  palette: {
    mode: 'dark',
    primary: { main: '#58a6ff' },
    secondary: { main: '#1f6feb' },
    background: {
      default: '#0d1117',
      paper: '#161b22',
    },
    text: {
      primary: '#f0f6fc',
      secondary: '#8b949e',
      disabled: '#484f58',
    },
    divider: '#30363d',
    success: { main: '#3fb950' },
    error: { main: '#f85149' },
    warning: { main: '#d29922' },
    info: { main: '#58a6ff' },
  },
  typography: {
    fontFamily: fonts.sans,
    h1: { fontFamily: fonts.sans, fontWeight: 800, letterSpacing: '-0.02em' },
    h2: { fontFamily: fonts.sans, fontWeight: 700, letterSpacing: '-0.02em' },
    h3: { fontFamily: fonts.sans, fontWeight: 700, letterSpacing: '-0.01em' },
    h4: { fontFamily: fonts.sans, fontWeight: 600, letterSpacing: '-0.01em' },
    h5: { fontFamily: fonts.sans, fontWeight: 600 },
    h6: { fontFamily: fonts.sans, fontWeight: 600 },
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
    primary: { main: '#0969da' },
    secondary: { main: '#0550ae' },
    background: {
      default: '#ffffff',
      paper: '#f6f8fa',
    },
    text: {
      primary: '#1f2328',
      secondary: '#656d76',
      disabled: '#6e7781',
    },
    divider: '#d0d7de',
    success: { main: '#1a7f37' },
    error: { main: '#cf222e' },
    warning: { main: '#9a6700' },
    info: { main: '#0969da' },
  },
  typography: {
    fontFamily: fonts.sans,
    h1: { fontFamily: fonts.sans, fontWeight: 800, letterSpacing: '-0.02em' },
    h2: { fontFamily: fonts.sans, fontWeight: 700, letterSpacing: '-0.02em' },
    h3: { fontFamily: fonts.sans, fontWeight: 700, letterSpacing: '-0.01em' },
    h4: { fontFamily: fonts.sans, fontWeight: 600, letterSpacing: '-0.01em' },
    h5: { fontFamily: fonts.sans, fontWeight: 600 },
    h6: { fontFamily: fonts.sans, fontWeight: 600 },
    subtitle1: { fontWeight: 600 },
    subtitle2: { fontWeight: 600 },
    body1: { fontWeight: 400, lineHeight: 1.7 },
    body2: { fontWeight: 400, lineHeight: 1.6 },
    caption: { fontWeight: 500, letterSpacing: '0.04em' },
    overline: { fontWeight: 500, letterSpacing: '0.15em', fontSize: '0.6875rem' },
    button: { fontWeight: 600 },
  },
});
