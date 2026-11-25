import { describe, expect, it } from 'vitest';

import { applyPatch, clamp, cycleSettingsSchema } from './settings';

const payload = {
  lengths: {
    auto: true,
    cycle_length: 29,
    period_length: 5,
    calculated: { cycle_length: 29, period_length: 5, based_on_cycles: 6 },
    manual: { cycle_length: null, period_length: null },
  },
  reminders: [{ code: 'before_period', enabled: true, time: '09:00', days_before: 2 }],
};

describe('cycleSettingsSchema', () => {
  it('maps the lengths and ignores the reminders (edited on /profile/notifications)', () => {
    const s = cycleSettingsSchema.parse(payload);
    expect(s.lengths).toMatchObject({ auto: true, cycleLength: 29, periodLength: 5 });
    expect(s.lengths.calculated.basedOnCycles).toBe(6);
    expect(s).not.toHaveProperty('reminders');
  });

  it('falls back to defaults for missing fields', () => {
    const s = cycleSettingsSchema.parse({});
    expect(s.lengths).toMatchObject({ auto: true, cycleLength: 28, periodLength: 5 });
    expect(s.lengths.calculated).toEqual({ cycleLength: null, periodLength: null, basedOnCycles: null });
  });
});

describe('applyPatch', () => {
  const base = cycleSettingsSchema.parse(payload);

  it('shows manual lengths when switched off, the medians when back on', () => {
    const manual = applyPatch(base, { lengths_auto: false, cycle_length: 31, period_length: 6 });
    expect(manual.lengths).toMatchObject({ auto: false, cycleLength: 31, periodLength: 6 });
    expect(manual.lengths.manual).toEqual({ cycleLength: 31, periodLength: 6 });
    const auto = applyPatch(manual, { lengths_auto: true });
    expect(auto.lengths).toMatchObject({ auto: true, cycleLength: 29, periodLength: 5 });
  });
});

describe('clamp', () => {
  it('clamps into a range', () => {
    expect(clamp(50, { min: 20, max: 45 })).toBe(45);
    expect(clamp(1, { min: 2, max: 10 })).toBe(2);
    expect(clamp(7, { min: 2, max: 10 })).toBe(7);
  });
});
