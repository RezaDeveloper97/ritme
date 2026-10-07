import { describe, expect, it } from 'vitest';

import { themeInitScript } from './init-script';
import { isThemePreference, resolveTheme } from './store';

describe('theme', () => {
  it('resolves system against the OS', () => {
    expect(resolveTheme('system', 'dark')).toBe('dark');
    expect(resolveTheme('light', 'dark')).toBe('light');
  });
  it('validates stored values', () => {
    expect(isThemePreference('dark')).toBe(true);
    expect(isThemePreference('blue')).toBe(false);
  });
  it('init script reads the same storage key and sets data-theme', () => {
    expect(themeInitScript).toContain('ritme_instructor_theme');
    expect(themeInitScript).toContain('d.dataset.theme=t');
  });
});
