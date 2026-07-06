'use client';

import { create } from 'zustand';

/**
 * Appearance preference. Client/UI state only (CLAUDE.md §8) — persisted to
 * localStorage and mirrored onto <html data-theme> so CSS variables switch.
 * 'system' follows the OS via prefers-color-scheme.
 */
export type ThemePreference = 'system' | 'light' | 'dark';

export const THEME_KEY = 'ritme_theme';

const isBrowser = (): boolean => typeof window !== 'undefined';

function resolve(pref: ThemePreference): 'light' | 'dark' {
  if (pref !== 'system') return pref;
  if (!isBrowser()) return 'light';
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

export function applyTheme(pref: ThemePreference): void {
  if (!isBrowser()) return;
  document.documentElement.dataset.theme = resolve(pref);
}

function readStored(): ThemePreference {
  if (!isBrowser()) return 'system';
  const raw = window.localStorage.getItem(THEME_KEY);
  return raw === 'light' || raw === 'dark' || raw === 'system' ? raw : 'system';
}

interface ThemeState {
  theme: ThemePreference;
  setTheme: (theme: ThemePreference) => void;
}

export const useThemeStore = create<ThemeState>((set) => ({
  theme: readStored(),
  setTheme: (theme) => {
    if (isBrowser()) window.localStorage.setItem(THEME_KEY, theme);
    applyTheme(theme);
    set({ theme });
  },
}));
