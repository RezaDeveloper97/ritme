import { describe, expect, it } from 'vitest';

import { activeKeys, highlightKey, patternTone, toggleKey, toneOf } from './symptoms';

const KEYS = ['a', 'b', 'c', 'd'];

describe('analysis-symptoms model', () => {
  it('defaults to the first two lanes and keeps survivors', () => {
    expect(activeKeys(KEYS, null)).toEqual(['a', 'b']);
    expect(activeKeys(KEYS, ['c', 'x'])).toEqual(['c']);
    expect(activeKeys(KEYS, ['x'])).toEqual(['a', 'b']);
  });

  it('toggles within 1–3 lanes in trend order', () => {
    expect(toggleKey(KEYS, ['a'], 'a')).toEqual(['a']);
    expect(toggleKey(KEYS, ['a', 'b'], 'a')).toEqual(['b']);
    expect(toggleKey(KEYS, ['c'], 'a')).toEqual(['a', 'c']);
    expect(toggleKey(KEYS, ['a', 'b', 'c'], 'd')).toEqual(['b', 'c', 'd']);
  });

  it('picks tones and highlight wording', () => {
    expect(toneOf(KEYS, 'b')).toBe('bloom');
    expect(patternTone(KEYS, 'b', 0)).toBe('bloom');
    expect(patternTone(KEYS, 'z', 0)).toBe('period');
    expect(highlightKey({ relation: 'before_period', startDay: 27, endDay: 28 })).toBe('before_period');
    expect(highlightKey({ relation: 'mid', startDay: 13, endDay: 13 })).toBe('mid_day');
    expect(highlightKey({ relation: null, startDay: 10, endDay: 12 })).toBe('mid');
  });
});
