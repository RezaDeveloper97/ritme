'use client';

import { create } from 'zustand';

/**
 * Appearance preference. Client/UI state only (CLAUDE.md §8) — persisted to
 * localStorage and mirrored onto <html data-theme> so CSS variables switch.
 * 'system' follows the OS via prefers-color-scheme.
 */
export type ThemePreference = 'system' | 'light' | 'dark';

/** What the preference actually resolves to once the OS is taken into account. */
export type ResolvedTheme = 'light' | 'dark';

export const THEME_KEY = 'ritme_theme';

export const THEME_PREFERENCES: readonly ThemePreference[] = ['system', 'light', 'dark'];

const isBrowser = (): boolean => typeof window !== 'undefined';

export function isThemePreference(value: unknown): value is ThemePreference {
  return value === 'system' || value === 'light' || value === 'dark';
}

/** 'system' asks the OS; an explicit choice is its own answer. */
export function resolveTheme(pref: ThemePreference): ResolvedTheme {
  if (pref !== 'system') return pref;
  if (!isBrowser()) return 'light';
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

/**
 * The browser chrome (status bar, address bar, PWA title bar) is painted from
 * `<meta name="theme-color">`, which cannot reference a CSS variable. The layout
 * ships one meta per `prefers-color-scheme` so the *first paint* is right; once
 * the user picks a theme that disagrees with the OS, both of them have to be
 * rewritten to the resolved --page, otherwise the OS keeps winning up there
 * while the page below has already flipped.
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
 * Writes the resolved theme onto <html>. `data-theme` drives the token
 * overrides in globals.css; `color-scheme` (set in CSS from the same attribute)
 * is what tells the browser to darken its own widgets — scrollbars, form
 * controls, the spellcheck underline — which no amount of CSS variables can do.
 */
export function applyTheme(pref: ThemePreference): ResolvedTheme {
  const resolved = resolveTheme(pref);
  if (!isBrowser()) return resolved;
  document.documentElement.dataset.theme = resolved;
  syncThemeColor();
  return resolved;
}

function readStored(): ThemePreference {
  if (!isBrowser()) return 'system';
  const raw = window.localStorage.getItem(THEME_KEY);
  return isThemePreference(raw) ? raw : 'system';
}

interface ThemeState {
  theme: ThemePreference;
  /** What `theme` currently resolves to — for UI that must know the real mode. */
  resolved: ResolvedTheme;
  setTheme: (theme: ThemePreference) => void;
  /** Re-resolves without changing the preference (OS flipped under 'system'). */
  syncResolved: () => void;
}

// Read once at module scope: the store is created on the client, and the
// inline bootstrap has already written the same value onto <html>, so the
// first render agrees with the DOM (no hydration mismatch).
const initial = readStored();

export const useThemeStore = create<ThemeState>((set, get) => ({
  theme: initial,
  resolved: resolveTheme(initial),
  setTheme: (theme) => {
    if (isBrowser()) window.localStorage.setItem(THEME_KEY, theme);
    set({ theme, resolved: applyTheme(theme) });
  },
  syncResolved: () => set({ resolved: applyTheme(get().theme) }),
}));
