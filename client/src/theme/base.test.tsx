import { afterEach, describe, expect, it } from 'vitest';
import { applyDocumentTheme } from './documentTheme';
import { themes } from './index';

const root = document.documentElement;

function resetThemeState() {
  root.classList.remove('dark');
  root.removeAttribute('data-theme-name');
  root.removeAttribute('data-theme-mode');
  root.removeAttribute('style');
}

describe('document theme bridge', () => {
  afterEach(() => {
    resetThemeState();
  });

  it('maps the dark primer palette into shadcn document variables', () => {
    applyDocumentTheme(themes.primer.dark, 'primer', 'dark');

    expect(root.dataset.themeName).toBe('primer');
    expect(root.dataset.themeMode).toBe('dark');
    expect(root.classList.contains('dark')).toBe(true);
    expect(root.style.getPropertyValue('--background')).toBe(themes.primer.dark.palette.background.default);
    expect(root.style.getPropertyValue('--card')).toBe(themes.primer.dark.palette.background.paper);
    expect(root.style.getPropertyValue('--primary')).toBe(themes.primer.dark.palette.primary.main);
    expect(root.style.getPropertyValue('--ring')).toBe(themes.primer.dark.palette.primary.main);
    expect(root.style.getPropertyValue('--font-sans')).toContain('Figtree');
    expect(root.style.getPropertyValue('--font-mono')).toContain('Source Code Pro');
  });

  it('updates mode-specific flags and typography when switching to solarized light', () => {
    applyDocumentTheme(themes.solarized.light, 'solarized', 'light');

    expect(root.dataset.themeName).toBe('solarized');
    expect(root.dataset.themeMode).toBe('light');
    expect(root.classList.contains('dark')).toBe(false);
    expect(root.style.getPropertyValue('--foreground')).toBe(themes.solarized.light.palette.text.primary);
    expect(root.style.getPropertyValue('--sidebar')).toBe(themes.solarized.light.palette.background.paper);
    expect(root.style.getPropertyValue('--destructive')).toBe(themes.solarized.light.palette.error.main);
    expect(root.style.getPropertyValue('--font-heading')).toContain('Newsreader');
    expect(root.style.getPropertyValue('--arsenale-primary-14')).not.toBe('');
  });
});
