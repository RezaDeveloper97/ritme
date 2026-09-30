import { describe, expect, it } from 'vitest';

import { applyPatch, displayClock, isClock, settingsSchema } from './settings';

const payload = {
  groups: [
    { code: 'cycle', items: [{ code: 'before_period', enabled: true }, { code: 'fertile_window', enabled: false }] },
    { code: 'health', items: [{ code: 'vitals', enabled: false }, { code: 'telepathy', enabled: true }] },
    { code: 'future', items: [{ code: 'x', enabled: true }] },
  ],
  quiet_hours: { enabled: true, start: '23:00', end: '08:00' },
  neutral_copy: true,
};

describe('settingsSchema', () => {
  it('keeps known groups and categories in server order', () => {
    const s = settingsSchema.parse(payload);
    expect(s.groups.map((g) => g.code)).toEqual(['cycle', 'health']);
    expect(s.groups[1].items).toEqual([{ code: 'vitals', enabled: false }]);
    expect(s.quietHours).toEqual({ enabled: true, start: '23:00', end: '08:00' });
    expect(s.neutralCopy).toBe(true);
  });

  it('falls back to the board defaults for missing fields', () => {
    const s = settingsSchema.parse({});
    expect(s.groups).toEqual([]);
    expect(s.quietHours).toEqual({ enabled: true, start: '23:00', end: '08:00' });
    expect(s.neutralCopy).toBe(true);
  });
});

describe('applyPatch', () => {
  const base = settingsSchema.parse(payload);

  it('flips only the patched category', () => {
    const next = applyPatch(base, { categories: { fertile_window: true } });
    expect(next.groups[0].items).toEqual([
      { code: 'before_period', enabled: true },
      { code: 'fertile_window', enabled: true },
    ]);
    expect(next.groups[1]).toEqual(base.groups[1]);
  });

  it('merges quiet hours and neutral copy', () => {
    const next = applyPatch(base, { quiet_hours: { start: '22:30' }, neutral_copy: false });
    expect(next.quietHours).toEqual({ enabled: true, start: '22:30', end: '08:00' });
    expect(next.neutralCopy).toBe(false);
    expect(base.neutralCopy).toBe(true);
  });
});

describe('clock helpers', () => {
  it('validates HH:MM', () => {
    expect(isClock('23:00')).toBe(true);
    expect(isClock('08:05')).toBe(true);
    expect(isClock('24:00')).toBe(false);
    expect(isClock('8:00')).toBe(false);
    expect(isClock('')).toBe(false);
  });

  it('drops the leading zero of the hour', () => {
    expect(displayClock('08:00')).toBe('8:00');
    expect(displayClock('23:00')).toBe('23:00');
    expect(displayClock('00:30')).toBe('0:30');
  });
});
