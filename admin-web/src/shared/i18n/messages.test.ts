import { describe, expect, it } from 'vitest';

import { UI_BUNDLES, uiDirection, resolveUiLocale } from './ui-locales';

function keysOf(obj: unknown, prefix = ''): string[] {
  if (typeof obj !== 'object' || obj === null) return [prefix];
  return Object.entries(obj).flatMap(([k, v]) => keysOf(v, prefix ? `${prefix}.${k}` : k));
}

describe('admin UI bundles', () => {
  it('every bundle has exactly the default bundle keys', () => {
    const reference = keysOf(UI_BUNDLES.fa).sort();
    for (const [code, bundle] of Object.entries(UI_BUNDLES)) {
      expect(keysOf(bundle).sort(), code).toEqual(reference);
    }
  });

  it('direction comes from the bundle meta', () => {
    expect(uiDirection('fa')).toBe('rtl');
    expect(uiDirection('en')).toBe('ltr');
    expect(resolveUiLocale('xx')).toBe('fa');
  });
});
