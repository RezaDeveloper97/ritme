'use client';

import { create } from 'zustand';

import { DEFAULT_THEME as FALLBACK_THEME, THEME_KEY as STORAGE_KEY } from './store-constants';

/**
 * Admin appearance: light or dark, per browser. Mirrors the user app's rule
 * (frontend/CLAUDE.md §10.3): one attribute, <html data-theme>, flips every
 * token in globals.css; nothing else branches on the theme.
 */
export type ThemePreference = 'light' | 'dark';

export const THEME_KEY = STORAGE_KEY;
export const DEFAULT_THEME: ThemePreference = FALLBACK_THEME;

export function isThemePreference(value: unknown): value is ThemePreference {
  return value === 'light' || value === 'dark';
}

export function applyTheme(theme: ThemePreference): void {
  if (typeof document === 'undefined') return;
  document.documentElement.dataset.theme = theme;
}

function readStored(): ThemePreference {
  if (typeof window === 'undefined') return DEFAULT_THEME;
  try {
    const raw = window.localStorage.getItem(THEME_KEY);
    return isThemePreference(raw) ? raw : DEFAULT_THEME;
  } catch {
    return DEFAULT_THEME;
  }
}

interface ThemeState {
  theme: ThemePreference;
  setTheme: (theme: ThemePreference) => void;
  toggle: () => void;
}

export const useThemeStore = create<ThemeState>((set, get) => ({
  theme: readStored(),
  setTheme: (theme) => {
    try {
      window.localStorage.setItem(THEME_KEY, theme);
    } catch {
      // Private mode: the choice lasts for this tab only.
    }
    applyTheme(theme);
    set({ theme });
  },
  toggle: () => get().setTheme(get().theme === 'dark' ? 'light' : 'dark'),
}));
