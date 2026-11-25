import { describe, expect, it } from 'vitest';

import { applyReminderPatch, clamp, remindersSchema } from './reminders';

const payload = {
  lengths: { auto: true, cycle_length: 29, period_length: 5 },
  reminders: [
    { code: 'before_period', enabled: true, time: '09:00', days_before: 2 },
    { code: 'pms', enabled: true, time: '09:00', cycle_day: 27 },
    { code: 'telepathy', enabled: true, time: '09:00' },
    { code: 'pill', enabled: false, time: '21:00' },
  ],
};

describe('remindersSchema', () => {
  it('keeps the known reminders of the cycle-settings payload', () => {
    const list = remindersSchema.parse(payload);
    expect(list.map((r) => r.code)).toEqual(['before_period', 'pms', 'pill']);
    expect(list[0].daysBefore).toBe(2);
    expect(list[1].cycleDay).toBe(27);
  });

  it('is empty when the payload has none', () => {
    expect(remindersSchema.parse({})).toEqual([]);
  });
});

describe('applyReminderPatch', () => {
  const base = remindersSchema.parse(payload);

  it('updates only the patched fields', () => {
    const next = applyReminderPatch(base, { before_period: { days_before: 3 }, pill: { enabled: true, time: '22:00' } });
    expect(next[0]).toMatchObject({ enabled: true, time: '09:00', daysBefore: 3 });
    expect(next[2]).toMatchObject({ enabled: true, time: '22:00' });
    expect(next[1]).toBe(base[1]);
  });
});

describe('clamp', () => {
  it('clamps into a range', () => {
    expect(clamp(9, { min: 1, max: 7 })).toBe(7);
    expect(clamp(0, { min: 1, max: 7 })).toBe(1);
  });
});
