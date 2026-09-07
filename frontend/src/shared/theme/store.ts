'use client';

import { create } from 'zustand';

/**
 * Appearance preference. Client/UI state only (CLAUDE.md §8) — persisted to
 * localStorage and mirrored onto <html data-theme> so CSS variables switch.
 *
 * There is deliberately no 'system' option: the app is a **light** app until
 * the user turns dark mode on themselves. Following the OS meant a user whose
 * phone happens to be in dark mode got a theme they never asked Ritme for, and
 * had no way to read the light one — the switch in Profile is the only thing
 * that changes it.
 */
export type ThemePreference = 'light' | 'dark';

export const THEME_KEY = 'ritme_theme';

/** The default a fresh install (or an unreadable stored value) lands on. */
export const DEFAULT_THEME: ThemePreference = 'light';

export const THEME_PREFERENCES: readonly ThemePreference[] = ['light', 'dark'];

const isBrowser = (): boolean => typeof window !== 'undefined';

export function isThemePreference(value: unknown): value is ThemePreference {
  return value === 'light' || value === 'dark';
}

/**
 * The browser chrome (status bar, address bar, PWA title bar) is painted from
 * `<meta name="theme-color">`, which cannot reference a CSS variable. The layout
 * ships the light value, so turning dark mode on has to rewrite the meta to the
 * new --page — otherwise the chrome stays light above a dark page.
 */
function syncThemeColor(): void {
  const page = getComputedStyle(document.documentElement).getPropertyValue('--page').trim();
  if (!page) return;
  document
    .querySelectorAll<HTMLMetaElement>('meta[name="theme-color"]')
    .forEach((meta) => {
      meta.content = page;
    });
}

/**
 * Writes the theme onto <html>. `data-theme` drives the token overrides in
 * globals.css; `color-scheme` (set in CSS from the same attribute) is what
 * tells the browser to darken its own widgets — scrollbars, form controls, the
 * spellcheck underline — which no amount of CSS variables can do.
 */
export function applyTheme(theme: ThemePreference): ThemePreference {
  if (!isBrowser()) return theme;
  document.documentElement.dataset.theme = theme;
  syncThemeColor();
  return theme;
}

function readStored(): ThemePreference {
  if (!isBrowser()) return DEFAULT_THEME;
  const raw = window.localStorage.getItem(THEME_KEY);
  return isThemePreference(raw) ? raw : DEFAULT_THEME;
}

interface ThemeState {
  theme: ThemePreference;
  setTheme: (theme: ThemePreference) => void;
}

// Read once at module scope: the store is created on the client, and the
// inline bootstrap has already written the same value onto <html>, so the
// first render agrees with the DOM (no hydration mismatch).
export const useThemeStore = create<ThemeState>((set) => ({
  theme: readStored(),
  setTheme: (theme) => {
    if (isBrowser()) window.localStorage.setItem(THEME_KEY, theme);
    applyTheme(theme);
    set({ theme });
  },
}));
