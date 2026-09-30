import { describe, expect, it } from 'vitest';

import { applyPatch, clamp, cycleSettingsSchema, displayClock, isClock } from './settings';

const payload = {
  lengths: {
    auto: true,
    cycle_length: 29,
    period_length: 5,
    calculated: { cycle_length: 29, period_length: 5, based_on_cycles: 6 },
    manual: { cycle_length: null, period_length: null },
  },
  reminders: [
    { code: 'before_period', enabled: true, time: '09:00', days_before: 2 },
    { code: 'pms', enabled: true, time: '09:00', cycle_day: 27 },
    { code: 'telepathy', enabled: true, time: '09:00' },
    { code: 'pill', enabled: false, time: '21:00' },
  ],
};

describe('cycleSettingsSchema', () => {
  it('maps the payload and drops unknown reminders', () => {
    const s = cycleSettingsSchema.parse(payload);
    expect(s.lengths).toMatchObject({ auto: true, cycleLength: 29, periodLength: 5 });
    expect(s.lengths.calculated.basedOnCycles).toBe(6);
    expect(s.reminders.map((r) => r.code)).toEqual(['before_period', 'pms', 'pill']);
    expect(s.reminders[0].daysBefore).toBe(2);
    expect(s.reminders[1].cycleDay).toBe(27);
  });

  it('falls back to defaults for missing fields', () => {
    const s = cycleSettingsSchema.parse({});
    expect(s.lengths).toMatchObject({ auto: true, cycleLength: 28, periodLength: 5 });
    expect(s.lengths.calculated).toEqual({ cycleLength: null, periodLength: null, basedOnCycles: null });
    expect(s.reminders).toEqual([]);
  });
});

describe('applyPatch', () => {
  const base = cycleSettingsSchema.parse(payload);

  it('updates only the patched reminder fields', () => {
    const next = applyPatch(base, { reminders: { before_period: { days_before: 3 }, pill: { enabled: true } } });
    expect(next.reminders[0]).toMatchObject({ enabled: true, time: '09:00', daysBefore: 3 });
    expect(next.reminders[2]).toMatchObject({ enabled: true, time: '21:00' });
    expect(next.reminders[1]).toBe(base.reminders[1]);
  });

  it('shows manual lengths when switched off, the medians when back on', () => {
    const manual = applyPatch(base, { lengths_auto: false, cycle_length: 31, period_length: 6 });
    expect(manual.lengths).toMatchObject({ auto: false, cycleLength: 31, periodLength: 6 });
    expect(manual.lengths.manual).toEqual({ cycleLength: 31, periodLength: 6 });
    const auto = applyPatch(manual, { lengths_auto: true });
    expect(auto.lengths).toMatchObject({ auto: true, cycleLength: 29, periodLength: 5 });
  });
});

describe('helpers', () => {
  it('validates and displays clocks', () => {
    expect(isClock('21:30')).toBe(true);
    expect(isClock('9:00')).toBe(false);
    expect(displayClock('09:00')).toBe('9:00');
  });

  it('clamps into a range', () => {
    expect(clamp(50, { min: 20, max: 45 })).toBe(45);
    expect(clamp(1, { min: 2, max: 10 })).toBe(2);
    expect(clamp(7, { min: 2, max: 10 })).toBe(7);
  });
});
