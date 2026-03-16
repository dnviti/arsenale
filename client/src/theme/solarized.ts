import { createTheme, type Theme } from '@mui/material';
import { sharedComponents, sharedShape } from './base';

const fonts = {
  heading: "'Newsreader', Georgia, serif",
  body: "'Libre Franklin', system-ui, sans-serif",
  mono: "'Inconsolata', 'JetBrains Mono', monospace",
};

export const dark: Theme = createTheme({
  shape: sharedShape,
  components: sharedComponents,
  palette: {
    mode: 'dark',
    primary: { main: '#2aa198' },
    secondary: { main: '#268bd2' },
    background: {
      default: '#002b36',
      paper: '#073642',
    },
    text: {
      primary: '#eee8d5',
      secondary: '#93a1a1',
      disabled: '#586e75',
    },
    divider: '#0d4a5a',
    success: { main: '#859900' },
    error: { main: '#dc322f' },
    warning: { main: '#b58900' },
    info: { main: '#268bd2' },
  },
  typography: {
    fontFamily: fonts.body,
    h1: { fontFamily: fonts.heading, fontWeight: 500, letterSpacing: '-0.02em' },
    h2: { fontFamily: fonts.heading, fontWeight: 500, letterSpacing: '-0.02em' },
    h3: { fontFamily: fonts.heading, fontWeight: 500, letterSpacing: '-0.01em' },
    h4: { fontFamily: fonts.heading, fontWeight: 500, letterSpacing: '-0.01em' },
    h5: { fontFamily: fonts.heading, fontWeight: 500 },
    h6: { fontFamily: fonts.heading, fontWeight: 500 },
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
    primary: { main: '#2aa198' },
    secondary: { main: '#268bd2' },
    background: {
      default: '#fdf6e3',
      paper: '#eee8d5',
    },
    text: {
      primary: '#073642',
      secondary: '#586e75',
      disabled: '#93a1a1',
    },
    divider: '#d6ceb5',
    success: { main: '#859900' },
    error: { main: '#dc322f' },
    warning: { main: '#b58900' },
    info: { main: '#268bd2' },
  },
  typography: {
    fontFamily: fonts.body,
    h1: { fontFamily: fonts.heading, fontWeight: 500, letterSpacing: '-0.02em' },
    h2: { fontFamily: fonts.heading, fontWeight: 500, letterSpacing: '-0.02em' },
    h3: { fontFamily: fonts.heading, fontWeight: 500, letterSpacing: '-0.01em' },
    h4: { fontFamily: fonts.heading, fontWeight: 500, letterSpacing: '-0.01em' },
    h5: { fontFamily: fonts.heading, fontWeight: 500 },
    h6: { fontFamily: fonts.heading, fontWeight: 500 },
    subtitle1: { fontWeight: 600 },
    subtitle2: { fontWeight: 600 },
    body1: { fontWeight: 400, lineHeight: 1.7 },
    body2: { fontWeight: 400, lineHeight: 1.6 },
    caption: { fontWeight: 500, letterSpacing: '0.04em' },
    overline: { fontWeight: 500, letterSpacing: '0.15em', fontSize: '0.6875rem' },
    button: { fontWeight: 600 },
  },
});
