import { describe, expect, it } from 'vitest';

import { bandEdges, dayTone } from './tones';

describe('dayTone', () => {
  it('splits logged and predicted periods', () => {
    expect(dayTone('period', true)).toBe('period');
    expect(dayTone('period', false)).toBe('predicted');
  });

  it('folds ovulation into the fertile band', () => {
    expect(dayTone('ovulation', false)).toBe('fertile');
    expect(dayTone('fertile', false)).toBe('fertile');
  });

  it('keeps PMS and plain days', () => {
    expect(dayTone('pms', false)).toBe('pms');
    expect(dayTone(null, false)).toBeNull();
    expect(dayTone(null, true)).toBe('period');
  });
});

describe('bandEdges', () => {
  it('merges runs of one tone into a single band per row', () => {
    const edges = bandEdges([null, 'pms', 'pms', 'pms', 'period', null, 'fertile']);
    expect(edges).toEqual([
      null,
      { start: true, end: false },
      { start: false, end: false },
      { start: false, end: true },
      { start: true, end: true },
      null,
      { start: true, end: true },
    ]);
  });

  it('closes a band at the row edge', () => {
    expect(bandEdges(['fertile', 'fertile'])).toEqual([
      { start: true, end: false },
      { start: false, end: true },
    ]);
  });
});
