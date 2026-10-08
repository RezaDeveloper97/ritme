import { describe, expect, it } from 'vitest';

import { chipMode } from './chip';

describe('chipMode (CB-NAV-03)', () => {
  it('maps the life stage, IVF as the TTC sub-mode', () => {
    expect(chipMode({ mode: 'cycle', ivfIui: false })).toBe('cycle');
    expect(chipMode({ mode: 'ttc', ivfIui: false })).toBe('ttc');
    expect(chipMode({ mode: 'ttc', ivfIui: true })).toBe('ivf');
    expect(chipMode({ mode: 'cycle', ivfIui: true })).toBe('cycle');
    expect(chipMode({ mode: 'menopause', ivfIui: false })).toBe('menopause');
    expect(chipMode({ mode: 'teen', ivfIui: false })).toBe('teen');
  });

  it('draws nothing while unknown or for a companion account', () => {
    expect(chipMode(null)).toBeNull();
    expect(chipMode(undefined)).toBeNull();
    expect(chipMode({ mode: 'cycle', companion: true, ivfIui: false })).toBeNull();
  });
});
