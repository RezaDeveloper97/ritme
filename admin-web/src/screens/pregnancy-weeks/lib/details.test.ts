import { describe, expect, it } from 'vitest';

import { blankOrNull, cleanTranslations, nextTaskKey } from './details';

describe('week details helpers', () => {
  it('generates the first free task key', () => {
    expect(nextTaskKey(12, [])).toBe('w12_1');
    expect(nextTaskKey(12, ['w12_1', 'w12_3'])).toBe('w12_2');
  });
  it('cleans values', () => {
    expect(cleanTranslations({ fa: '', en: ' a' })).toEqual({ en: 'a' });
    expect(blankOrNull('  ')).toBeNull();
    expect(blankOrNull(' 1.6 ')).toBe('1.6');
  });
});
