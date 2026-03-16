import { createTheme, type Theme } from '@mui/material';
import { sharedComponents, sharedShape } from './base';

const fonts = {
  heading: "'Bricolage Grotesque', system-ui, sans-serif",
  body: "'Karla', system-ui, sans-serif",
  mono: "'JetBrains Mono', 'Fira Code', monospace",
};

export const dark: Theme = createTheme({
  shape: sharedShape,
  components: sharedComponents,
  palette: {
    mode: 'dark',
    primary: { main: '#61afef' },
    secondary: { main: '#528bcc' },
    background: {
      default: '#282c34',
      paper: '#21252b',
    },
    text: {
      primary: '#abb2bf',
      secondary: '#636d83',
      disabled: '#5c6370',
    },
    divider: '#3e4451',
    success: { main: '#98c379' },
    error: { main: '#e06c75' },
    warning: { main: '#e5c07b' },
    info: { main: '#56b6c2' },
  },
  typography: {
    fontFamily: fonts.body,
    h1: { fontFamily: fonts.heading, fontWeight: 700, letterSpacing: '-0.02em' },
    h2: { fontFamily: fonts.heading, fontWeight: 700, letterSpacing: '-0.02em' },
    h3: { fontFamily: fonts.heading, fontWeight: 600, letterSpacing: '-0.01em' },
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
    primary: { main: '#4078f2' },
    secondary: { main: '#3568d4' },
    background: {
      default: '#fafafa',
      paper: '#f0f0f0',
    },
    text: {
      primary: '#383a42',
      secondary: '#696c77',
      disabled: '#a0a1a7',
    },
    divider: '#d4d4d4',
    success: { main: '#50a14f' },
    error: { main: '#e45649' },
    warning: { main: '#c18401' },
    info: { main: '#0184bc' },
  },
  typography: {
    fontFamily: fonts.body,
    h1: { fontFamily: fonts.heading, fontWeight: 700, letterSpacing: '-0.02em' },
    h2: { fontFamily: fonts.heading, fontWeight: 700, letterSpacing: '-0.02em' },
    h3: { fontFamily: fonts.heading, fontWeight: 600, letterSpacing: '-0.01em' },
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
