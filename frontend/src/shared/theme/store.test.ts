import { afterEach, describe, expect, it, vi } from 'vitest';

import {
  DEFAULT_THEME,
  isThemePreference,
  resolveTheme,
  THEME_PREFERENCES,
  themeInitScript,
} from './index';

describe('isThemePreference', () => {
  it('accepts light, dark and system and nothing else', () => {
    expect(isThemePreference('light')).toBe(true);
    expect(isThemePreference('dark')).toBe(true);
    expect(isThemePreference('system')).toBe(true);
    // A stale or hand-edited localStorage value must not become the theme.
    expect(isThemePreference('Dark')).toBe(false);
    expect(isThemePreference(null)).toBe(false);
    expect(isThemePreference(undefined)).toBe(false);
    expect(isThemePreference('')).toBe(false);
  });
});

describe('the default', () => {
  it('follows the system, so a fresh install matches the phone', () => {
    expect(DEFAULT_THEME).toBe('system');
  });

  it('is one of the offered preferences', () => {
    expect(THEME_PREFERENCES).toContain(DEFAULT_THEME);
  });
});

describe('resolveTheme', () => {
  it('keeps an explicit choice whatever the OS says', () => {
    expect(resolveTheme('light', 'dark')).toBe('light');
    expect(resolveTheme('dark', 'light')).toBe('dark');
  });

  it('follows the OS for system', () => {
    expect(resolveTheme('system', 'dark')).toBe('dark');
    expect(resolveTheme('system', 'light')).toBe('light');
  });
});

// The inline bootstrap is a hand-written duplicate of resolveTheme; run it
// against a fake DOM to prove the two agree.
describe('themeInitScript', () => {
  afterEach(() => vi.unstubAllGlobals());

  function run(stored: string | null, osDark: boolean): string | undefined {
    const root = { dataset: {} as Record<string, string> };
    vi.stubGlobal('localStorage', { getItem: () => stored });
    vi.stubGlobal('window', { matchMedia: () => ({ matches: osDark }) });
    vi.stubGlobal('document', { documentElement: root, querySelectorAll: () => [] });
    vi.stubGlobal('getComputedStyle', () => ({ getPropertyValue: () => '' }));
    new Function(themeInitScript)();
    return root.dataset.theme;
  }

  it('paints the stored explicit choice', () => {
    expect(run('dark', false)).toBe('dark');
    expect(run('light', true)).toBe('light');
  });

  it('follows the OS for system, nothing stored, or junk', () => {
    for (const stored of ['system', null, 'Dark']) {
      expect(run(stored, true)).toBe('dark');
      expect(run(stored, false)).toBe('light');
    }
  });
});
