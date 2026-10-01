import { describe, expect, it } from 'vitest';

import { isPairGroup, isPhaseGroup, PAIR_STYLE, strengthTone } from './correlations';

describe('analysis-correlations model', () => {
  it('styles pairs and strengths', () => {
    expect(PAIR_STYLE.sleep_mood).toEqual({ icon: 'bed', tone: 'brand' });
    expect(strengthTone('strong')).toBe('data');
    expect(strengthTone('weak')).toBe('warm');
    expect(strengthTone(null)).toBe('warm');
  });

  it('tells group kinds apart', () => {
    expect(isPhaseGroup('luteal')).toBe(true);
    expect(isPhaseGroup('exercise')).toBe(false);
    expect(isPairGroup('sleep_under_6')).toBe(true);
    expect(isPairGroup('other')).toBe(false);
  });
});
