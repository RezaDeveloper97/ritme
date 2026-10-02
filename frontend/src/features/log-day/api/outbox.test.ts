import { describe, expect, it } from 'vitest';

import { logDayOutboxKey, mergeQueuedDay } from './outbox';

describe('log day outbox', () => {
  it('keys one entry per day', () => {
    expect(logDayOutboxKey('2026-10-02')).toBe('log-day:2026-10-02');
  });

  it('merges a newer partial save into the queued one', () => {
    const merged = mergeQueuedDay(
      { categories: { symptoms: { general: { hot_flashes: { level: 'mild', score: null } } }, menopause: { triggers: ['stress'] } } },
      { categories: { menopause: { triggers: null }, bleeding: { presence: 'none' } } },
    );
    expect(merged).toEqual({
      categories: {
        symptoms: { general: { hot_flashes: { level: 'mild', score: null } } },
        menopause: { triggers: null },
        bleeding: { presence: 'none' },
      },
    });
  });

  it('drops a voice mark the newer save overrides by hand', () => {
    const merged = mergeQueuedDay(
      { categories: { mood: { feelings: ['calm'] } }, voice_params: ['mood.feelings'] },
      { categories: { mood: { feelings: ['sad'] } } },
    );
    expect(merged.voice_params).toBeUndefined();
    expect(mergeQueuedDay(null, { categories: {} })).toEqual({ categories: {} });
  });
});
