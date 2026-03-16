import { createTheme, type Theme } from '@mui/material';
import { sharedComponents, sharedShape } from './base';

const fonts = {
  heading: "'Manrope', system-ui, sans-serif",
  body: "'Urbanist', system-ui, sans-serif",
  mono: "'Fira Code', 'JetBrains Mono', monospace",
};

export const dark: Theme = createTheme({
  shape: sharedShape,
  components: sharedComponents,
  palette: {
    mode: 'dark',
    primary: { main: '#7c3aed' },
    secondary: { main: '#fc6d26' },
    background: {
      default: '#0e0e1a',
      paper: '#14142b',
    },
    text: {
      primary: '#ededf0',
      secondary: '#9e9eb8',
      disabled: '#5a5a7a',
    },
    divider: '#2d2d52',
    success: { main: '#2da160' },
    error: { main: '#ec5941' },
    warning: { main: '#fc6d26' },
    info: { main: '#a78bfa' },
  },
  typography: {
    fontFamily: fonts.body,
    h1: { fontFamily: fonts.heading, fontWeight: 800, letterSpacing: '-0.02em' },
    h2: { fontFamily: fonts.heading, fontWeight: 700, letterSpacing: '-0.02em' },
    h3: { fontFamily: fonts.heading, fontWeight: 700, letterSpacing: '-0.01em' },
    h4: { fontFamily: fonts.heading, fontWeight: 600, letterSpacing: '-0.01em' },
    h5: { fontFamily: fonts.heading, fontWeight: 600 },
    h6: { fontFamily: fonts.heading, fontWeight: 600 },
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
    primary: { main: '#6e49cb' },
    secondary: { main: '#e24329' },
    background: {
      default: '#fafafa',
      paper: '#ffffff',
    },
    text: {
      primary: '#1e1e2e',
      secondary: '#55556e',
      disabled: '#8e8ea8',
    },
    divider: '#dcdbe5',
    success: { main: '#217645' },
    error: { main: '#c91c1c' },
    warning: { main: '#e24329' },
    info: { main: '#8b5cf6' },
  },
  typography: {
    fontFamily: fonts.body,
    h1: { fontFamily: fonts.heading, fontWeight: 800, letterSpacing: '-0.02em' },
    h2: { fontFamily: fonts.heading, fontWeight: 700, letterSpacing: '-0.02em' },
    h3: { fontFamily: fonts.heading, fontWeight: 700, letterSpacing: '-0.01em' },
    h4: { fontFamily: fonts.heading, fontWeight: 600, letterSpacing: '-0.01em' },
    h5: { fontFamily: fonts.heading, fontWeight: 600 },
    h6: { fontFamily: fonts.heading, fontWeight: 600 },
    subtitle1: { fontWeight: 600 },
    subtitle2: { fontWeight: 600 },
    body1: { fontWeight: 400, lineHeight: 1.7 },
    body2: { fontWeight: 400, lineHeight: 1.6 },
    caption: { fontWeight: 500, letterSpacing: '0.04em' },
    overline: { fontWeight: 500, letterSpacing: '0.15em', fontSize: '0.6875rem' },
    button: { fontWeight: 600 },
  },
});
