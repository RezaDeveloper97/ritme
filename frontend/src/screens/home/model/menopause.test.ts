import { describe, expect, it } from 'vitest';

import { flashElapsedSeconds, listedMessages, routedLink, scoreTrend } from './menopause';

describe('menopause home helpers', () => {
  it('keeps only links whose screen exists', () => {
    expect(routedLink('/checkups/3')).toBe('/checkups/3');
    expect(routedLink('/checkups')).toBe('/checkups');
    expect(routedLink('/menopause/score')).toBe('/menopause/score');
    expect(routedLink('/menopause/alert')).toBe('/menopause/alert');
    expect(routedLink('/menopause/treatment')).toBeNull();
    expect(routedLink('https://example.com/checkups/3')).toBeNull();
    expect(routedLink(null)).toBeNull();
  });

  it('reads the score delta as better / worse / same', () => {
    expect(scoreTrend(-4)).toEqual({ kind: 'better', points: 4 });
    expect(scoreTrend(5)).toEqual({ kind: 'worse', points: 5 });
    expect(scoreTrend(0)).toEqual({ kind: 'same' });
    expect(scoreTrend(null)).toBeNull();
  });

  it('drops the bleeding alert and the hero tip from the list', () => {
    const base = { priority: 'low' as const, title: 'T', body: null, action: null, link: null };
    const list = listedMessages(
      [
        { ...base, key: 'postmenopausal_bleeding', kind: 'alert' },
        { ...base, key: 'checkup_due', kind: 'reminder' },
        { ...base, key: 'stage_meno', kind: 'tip' },
        { ...base, key: 'sleep_tip', kind: 'tip' },
      ],
      'stage_meno',
    );
    expect(list.map((m) => m.key)).toEqual(['checkup_due', 'sleep_tip']);
  });

  it('caps the running timer at an hour', () => {
    expect(flashElapsedSeconds(100, 2500)).toBe(102);
    expect(flashElapsedSeconds(3590, 60_000)).toBe(3600);
  });
});
