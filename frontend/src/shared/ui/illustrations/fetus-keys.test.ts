import { describe, expect, it } from 'vitest';

import { FETUS_DRAWINGS, FETUS_ILLUSTRATION_KEYS, fetusDrawing, isFetusIllustrationKey } from './fetus-keys';

describe('fetus illustration keys', () => {
  it('has exactly one drawing per key (the admin picker list)', () => {
    expect(Object.keys(FETUS_DRAWINGS).sort()).toEqual([...FETUS_ILLUSTRATION_KEYS].sort());
    expect(new Set(FETUS_ILLUSTRATION_KEYS).size).toBe(FETUS_ILLUSTRATION_KEYS.length);
  });

  it('uses snake_case keys the admin API can validate', () => {
    for (const key of FETUS_ILLUSTRATION_KEYS) expect(key).toMatch(/^[a-z]+(_[a-z]+)*$/);
  });

  it('falls back to the neutral fetus for unknown / missing keys', () => {
    expect(isFetusIllustrationKey('raspberry')).toBe(true);
    expect(isFetusIllustrationKey('dragon_fruit')).toBe(false);
    expect(fetusDrawing('dragon_fruit')).toEqual(FETUS_DRAWINGS.fetus);
    expect(fetusDrawing(null)).toEqual(FETUS_DRAWINGS.fetus);
    expect(fetusDrawing('raspberry')).toEqual({ shape: 'cluster', tint: 'berry', size: 'm' });
  });
});
