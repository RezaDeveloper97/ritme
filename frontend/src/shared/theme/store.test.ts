import { afterEach, describe, expect, it, vi } from 'vitest';

import { isThemePreference, resolveTheme } from './store';

/**
 * Stand in for a browser whose OS is set to `prefersDark`. The module reads
 * `window` at call time, so it is enough to define it around the assertion.
 */
function withOs(prefersDark: boolean, run: () => void) {
  vi.stubGlobal('window', {
    matchMedia: (query: string) => ({
      matches: query.includes('dark') && prefersDark,
    }),
  });
  run();
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('isThemePreference', () => {
  it('accepts the three preferences and nothing else', () => {
    expect(isThemePreference('system')).toBe(true);
    expect(isThemePreference('light')).toBe(true);
    expect(isThemePreference('dark')).toBe(true);
    // A stale or hand-edited localStorage value must not become the theme.
    expect(isThemePreference('Dark')).toBe(false);
    expect(isThemePreference(null)).toBe(false);
    expect(isThemePreference(undefined)).toBe(false);
    expect(isThemePreference('')).toBe(false);
  });
});

describe('resolveTheme', () => {
  it('takes an explicit choice at its word, whatever the OS says', () => {
    withOs(true, () => {
      expect(resolveTheme('light')).toBe('light');
      expect(resolveTheme('dark')).toBe('dark');
    });
    withOs(false, () => {
      expect(resolveTheme('light')).toBe('light');
      expect(resolveTheme('dark')).toBe('dark');
    });
  });

  it('follows the OS under "system"', () => {
    withOs(true, () => expect(resolveTheme('system')).toBe('dark'));
    withOs(false, () => expect(resolveTheme('system')).toBe('light'));
  });

  it('resolves to light on the server, where there is no OS to ask', () => {
    // The layout renders before hydration; guessing dark there would flash.
    expect(resolveTheme('system')).toBe('light');
  });
});
