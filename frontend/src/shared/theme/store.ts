'use client';

import { create } from 'zustand';

/**
 * Appearance. Client/UI state only (CLAUDE.md §8) — the *preference* is
 * persisted to localStorage; the *resolved* theme is mirrored onto
 * <html data-theme> so the CSS variables switch.
 *
 * Night & Bloom ships every screen in light and dark, plus "follow the system"
 * (bloom/README.md decisions). A fresh install follows the OS; a light/dark
 * choice a user already stored is kept (bloom/QUESTIONS.md #2).
 */
export type ThemePreference = 'light' | 'dark' | 'system';
/** What is actually painted — `system` resolved against the OS. */
export type ResolvedTheme = 'light' | 'dark';

export const THEME_KEY = 'ritme_theme';

/** The default a fresh install (or an unreadable stored value) lands on. */
export const DEFAULT_THEME: ThemePreference = 'system';

export const THEME_PREFERENCES: readonly ThemePreference[] = ['light', 'dark', 'system'];

const DARK_QUERY = '(prefers-color-scheme: dark)';

const isBrowser = (): boolean => typeof window !== 'undefined';

export function isThemePreference(value: unknown): value is ThemePreference {
  return value === 'light' || value === 'dark' || value === 'system';
}

/** The OS appearance right now (light when it cannot be read, e.g. on the server). */
export function systemTheme(): ResolvedTheme {
  if (!isBrowser() || typeof window.matchMedia !== 'function') return 'light';
  return window.matchMedia(DARK_QUERY).matches ? 'dark' : 'light';
}

export function resolveTheme(
  preference: ThemePreference,
  system: ResolvedTheme = systemTheme(),
): ResolvedTheme {
  return preference === 'system' ? system : preference;
}

/**
 * The browser chrome (status bar, address bar, PWA title bar) is painted from
 * `<meta name="theme-color">`, which cannot reference a CSS variable, so a
 * theme switch rewrites it to the new --page (#F7F3FF / #17112B).
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
 * overrides in globals.css; `color-scheme` (set in CSS from the same
 * attribute) darkens the browser's own widgets.
 */
export function applyTheme(theme: ResolvedTheme): ResolvedTheme {
  if (!isBrowser()) return theme;
  document.documentElement.dataset.theme = theme;
  syncThemeColor();
  return theme;
}

function readStored(): ThemePreference {
  if (!isBrowser()) return DEFAULT_THEME;
  try {
    const raw = window.localStorage.getItem(THEME_KEY);
    return isThemePreference(raw) ? raw : DEFAULT_THEME;
  } catch {
    return DEFAULT_THEME;
  }
}

interface ThemeState {
  /** The stored choice: light, dark or follow-the-system. */
  preference: ThemePreference;
  /**
   * The painted theme. Kept under the historic name so existing light/dark
   * toggles (`theme === 'dark'`, `setTheme(...)`) keep working unchanged.
   */
  theme: ResolvedTheme;
  setPreference: (preference: ThemePreference) => void;
  /** Explicit light/dark — what a one-tap toggle means. */
  setTheme: (theme: ResolvedTheme) => void;
  /** Re-resolve after the OS appearance changed (only matters for `system`). */
  syncSystem: () => void;
}

const initialPreference = readStored();

// Read once at module scope: the store is created on the client, and the
// inline bootstrap has already written the same resolution onto <html>, so the
// first render agrees with the DOM (no hydration mismatch).
export const useThemeStore = create<ThemeState>((set, get) => ({
  preference: initialPreference,
  theme: resolveTheme(initialPreference),
  setPreference: (preference) => {
    if (isBrowser()) {
      try {
        window.localStorage.setItem(THEME_KEY, preference);
      } catch {
        // Private mode / blocked storage: the choice still applies this session.
      }
    }
    const theme = applyTheme(resolveTheme(preference));
    set({ preference, theme });
  },
  setTheme: (theme) => get().setPreference(theme),
  syncSystem: () => {
    if (get().preference !== 'system') return;
    set({ theme: applyTheme(systemTheme()) });
  },
}));

/**
 * Subscribes to OS appearance changes; returns the unsubscribe. Used by
 * ThemeApplier so a `system` user's app flips the moment the phone does.
 */
export function watchSystemTheme(onChange: () => void): () => void {
  if (!isBrowser() || typeof window.matchMedia !== 'function') return () => {};
  const media = window.matchMedia(DARK_QUERY);
  media.addEventListener('change', onChange);
  return () => media.removeEventListener('change', onChange);
}
