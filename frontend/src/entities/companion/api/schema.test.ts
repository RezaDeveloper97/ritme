import { describe, expect, it } from 'vitest';

import { readGoals } from './schema';

describe('readGoals', () => {
  it('reads the multi-select list in order, dropping unknown codes, repeats and «unknown»', () => {
    expect(readGoals(['support', 'cycle', 'romance', 'support', 'unknown'], 'support')).toEqual(['support', 'cycle']);
    expect(readGoals([], 'support')).toEqual([]);
  });

  it('falls back to the single goal of a server older than the multi-select', () => {
    expect(readGoals(undefined, 'low_energy')).toEqual(['low_energy']);
    expect(readGoals(undefined, 'unknown')).toEqual([]);
    expect(readGoals(undefined, undefined)).toEqual([]);
  });
});
