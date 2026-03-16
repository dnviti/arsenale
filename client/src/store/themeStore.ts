import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { ThemeMode, ThemeName } from '../theme/index';

interface ThemeState {
  themeName: ThemeName;
  mode: ThemeMode;
  setTheme: (name: ThemeName) => void;
  toggle: () => void;
  setMode: (mode: ThemeMode) => void;
}

const getSystemPreference = (): ThemeMode =>
  window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';

export const useThemeStore = create<ThemeState>()(
  persist(
    (set) => ({
      themeName: 'editorial',
      mode: getSystemPreference(),
      setTheme: (name: ThemeName) => set({ themeName: name }),
      toggle: () =>
        set((state) => ({ mode: state.mode === 'dark' ? 'light' : 'dark' })),
      setMode: (mode: ThemeMode) => set({ mode }),
    }),
    { name: 'arsenale-theme' }
  )
);
