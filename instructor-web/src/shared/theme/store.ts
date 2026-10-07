'use client';

import { create } from 'zustand';

import { DEFAULT_THEME as FALLBACK, THEME_KEY as STORAGE_KEY } from './store-constants';

/**
 * Appearance: light, dark or follow the system, per browser. One attribute,
 * <html data-theme>, flips every token in globals.css; nothing else branches on
 * the theme (frontend/CLAUDE.md §10.3).
 */
export type ThemePreference = 'light' | 'dark' | 'system';
export type ResolvedTheme = 'light' | 'dark';

export const THEME_KEY = STORAGE_KEY;
export const DEFAULT_THEME: ThemePreference = FALLBACK;

export function isThemePreference(value: unknown): value is ThemePreference {
  return value === 'light' || value === 'dark' || value === 'system';
}

export function systemTheme(): ResolvedTheme {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return 'light';
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

export function resolveTheme(pref: ThemePreference, system: ResolvedTheme = systemTheme()): ResolvedTheme {
  return pref === 'system' ? system : pref;
}

export function applyTheme(pref: ThemePreference): ResolvedTheme {
  const resolved = resolveTheme(pref);
  if (typeof document !== 'undefined') document.documentElement.dataset.theme = resolved;
  return resolved;
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
  preference: ThemePreference;
  setPreference: (pref: ThemePreference) => void;
  /** Flip what is painted right now (an explicit choice; leaves `system`). */
  toggle: () => void;
}

export const useThemeStore = create<ThemeState>((set, get) => ({
  preference: readStored(),
  setPreference: (preference) => {
    try {
      window.localStorage.setItem(THEME_KEY, preference);
    } catch {
      // private mode: the choice lasts for this tab only
    }
    applyTheme(preference);
    set({ preference });
  },
  toggle: () => get().setPreference(resolveTheme(get().preference) === 'dark' ? 'light' : 'dark'),
}));
