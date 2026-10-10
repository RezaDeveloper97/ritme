import { describe, expect, it } from 'vitest';

import { pbacScore } from '../model/pbac';
import { pbacDaySchema } from './pbac';

describe('pbacDaySchema', () => {
  it('parses the day, its period and the alert (golden pbac_flow)', () => {
    const day = pbacDaySchema.parse({
      date: '2026-09-22',
      light: 1,
      medium: 2,
      heavy: 3,
      clots: 'none',
      flooding: false,
      score: 71,
      period: { start: '2026-09-21', end: '2026-09-30', day: 2, score: 131, days_logged: 2 },
      alert_threshold: 100,
      alert: { code: 'pbac_over_100', title: 'Over 100', body: 'Tell your doctor', meta: { severity: 'caution' } },
    });
    expect(day).toEqual({
      date: '2026-09-22',
      light: 1,
      medium: 2,
      heavy: 3,
      clots: 'none',
      flooding: false,
      score: 71,
      period: { start: '2026-09-21', end: '2026-09-30', day: 2, score: 131, daysLogged: 2 },
      alertThreshold: 100,
      alert: { code: 'pbac_over_100', title: 'Over 100', body: 'Tell your doctor' },
    });
  });

  it('is lenient on an empty day and malformed fields', () => {
    const day = pbacDaySchema.parse({ date: '2026-09-22', light: -3, clots: 'huge', period: { day: 0 }, alert: 'x' });
    expect(day).toMatchObject({ light: 0, medium: 0, heavy: 0, clots: null, flooding: false, period: null, alert: null });
    expect(day.alertThreshold).toBe(100);
  });
});

describe('pbacScore', () => {
  it('matches the API formula', () => {
    expect(pbacScore({ light: 1, medium: 2, heavy: 3, clots: 'none', flooding: false })).toBe(71);
    expect(pbacScore({ light: 0, medium: 2, heavy: 2, clots: 'large', flooding: true })).toBe(60);
    expect(pbacScore({ light: 0, medium: 0, heavy: 0, clots: 'small', flooding: false })).toBe(1);
    expect(pbacScore({ light: 0, medium: 0, heavy: 0, clots: null, flooding: false })).toBe(0);
  });
});
