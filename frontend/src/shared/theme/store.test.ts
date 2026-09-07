import { describe, expect, it } from 'vitest';

import { DEFAULT_THEME, isThemePreference, THEME_PREFERENCES } from './store';

describe('isThemePreference', () => {
  it('accepts the two preferences and nothing else', () => {
    expect(isThemePreference('light')).toBe(true);
    expect(isThemePreference('dark')).toBe(true);
    // A stale or hand-edited localStorage value must not become the theme.
    expect(isThemePreference('Dark')).toBe(false);
    expect(isThemePreference(null)).toBe(false);
    expect(isThemePreference(undefined)).toBe(false);
    expect(isThemePreference('')).toBe(false);
  });

  // 'system' was removed on purpose: the OS never decides this app's theme.
  // If it ever comes back as a stored value, it must read as unknown and fall
  // back to light rather than quietly following the phone again.
  it('rejects the retired "system" value', () => {
    expect(isThemePreference('system')).toBe(false);
  });
});

describe('the default', () => {
  it('is light, so a dark phone does not darken the app on its own', () => {
    expect(DEFAULT_THEME).toBe('light');
  });

  it('is one of the offered preferences', () => {
    expect(THEME_PREFERENCES).toContain(DEFAULT_THEME);
  });
});
