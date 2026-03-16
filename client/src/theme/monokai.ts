import { createTheme, type Theme } from '@mui/material';
import { sharedComponents, sharedShape } from './base';

const fonts = {
  heading: "'Syne', system-ui, sans-serif",
  body: "'Rubik', system-ui, sans-serif",
  mono: "'Space Mono', 'Fira Code', monospace",
};

export const dark: Theme = createTheme({
  shape: sharedShape,
  components: sharedComponents,
  palette: {
    mode: 'dark',
    primary: { main: '#a6e22e' },
    secondary: { main: '#f92672' },
    background: {
      default: '#1e1f1c',
      paper: '#272822',
    },
    text: {
      primary: '#f8f8f2',
      secondary: '#b8b8a8',
      disabled: '#75715e',
    },
    divider: '#454640',
    success: { main: '#a6e22e' },
    error: { main: '#fd971f' },
    warning: { main: '#e6db74' },
    info: { main: '#66d9ef' },
  },
  typography: {
    fontFamily: fonts.body,
    h1: { fontFamily: fonts.heading, fontWeight: 800, letterSpacing: '-0.02em' },
    h2: { fontFamily: fonts.heading, fontWeight: 700, letterSpacing: '-0.02em' },
    h3: { fontFamily: fonts.heading, fontWeight: 700, letterSpacing: '-0.01em' },
    h4: { fontFamily: fonts.heading, fontWeight: 600, letterSpacing: '-0.01em' },
    h5: { fontFamily: fonts.heading, fontWeight: 600 },
    h6: { fontFamily: fonts.heading, fontWeight: 600 },
    subtitle1: { fontWeight: 500 },
    subtitle2: { fontWeight: 500 },
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
    primary: { main: '#6d8c14' },
    secondary: { main: '#d4145a' },
    background: {
      default: '#fdf8ee',
      paper: '#faf4e4',
    },
    text: {
      primary: '#2e2f2a',
      secondary: '#6b6c60',
      disabled: '#9e9e8a',
    },
    divider: '#d9d3c3',
    success: { main: '#6d8c14' },
    error: { main: '#c87010' },
    warning: { main: '#9e8e20' },
    info: { main: '#1a8a9e' },
  },
  typography: {
    fontFamily: fonts.body,
    h1: { fontFamily: fonts.heading, fontWeight: 800, letterSpacing: '-0.02em' },
    h2: { fontFamily: fonts.heading, fontWeight: 700, letterSpacing: '-0.02em' },
    h3: { fontFamily: fonts.heading, fontWeight: 700, letterSpacing: '-0.01em' },
    h4: { fontFamily: fonts.heading, fontWeight: 600, letterSpacing: '-0.01em' },
    h5: { fontFamily: fonts.heading, fontWeight: 600 },
    h6: { fontFamily: fonts.heading, fontWeight: 600 },
    subtitle1: { fontWeight: 500 },
    subtitle2: { fontWeight: 500 },
    body1: { fontWeight: 400, lineHeight: 1.7 },
    body2: { fontWeight: 400, lineHeight: 1.6 },
    caption: { fontWeight: 500, letterSpacing: '0.04em' },
    overline: { fontWeight: 500, letterSpacing: '0.15em', fontSize: '0.6875rem' },
    button: { fontWeight: 600 },
  },
});
