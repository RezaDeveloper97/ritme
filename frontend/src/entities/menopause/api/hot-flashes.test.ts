import { describe, expect, it } from 'vitest';

import { menopauseFlashDaySchema, toHotFlashDetailsBody } from './hot-flashes';
import { menopauseFlashSchema } from './schema';

// Trimmed from backend-go/contract/golden/menopause/flash_flow.en (CB-MENO-02) — re-copy when it changes.
const FLASH_DAY: unknown = {
  date: '2026-09-23',
  count: 1,
  night_count: 0,
  avg_duration_s: 102,
  running: null,
  items: [
    {
      id: 1,
      started_at: '2026-09-23T10:00:00+03:30',
      running: false,
      duration_s: 102,
      elapsed_s: 102,
      severity: 'severe',
      night: false,
      sweat: true,
      triggers: ['caffeine', 'warm_room'],
    },
  ],
};

describe('hot-flash day (CB-MENO-07)', () => {
  it('parses the day list with tiles and details', () => {
    const day = menopauseFlashDaySchema.parse(FLASH_DAY);
    expect(day).toMatchObject({ date: '2026-09-23', count: 1, nightCount: 0, avgDurationS: 102, running: null });
    expect(day.items[0]).toMatchObject({ id: 1, durationS: 102, severity: 'severe', sweat: true, triggers: ['caffeine', 'warm_room'] });
  });

  it('parses a running timer with no details and drops unknown trigger codes', () => {
    const flash = menopauseFlashSchema.parse({
      id: 2,
      started_at: '2026-09-23T10:00:00+03:30',
      running: true,
      duration_s: null,
      elapsed_s: 60,
      severity: 'bogus',
      triggers: ['stress', 'moonlight'],
    });
    expect(flash).toMatchObject({ running: true, severity: null, night: false, sweat: false, triggers: ['stress'] });
  });

  it('drops a malformed flash instead of failing the day', () => {
    const day = menopauseFlashDaySchema.parse({ date: '2026-09-23', items: [{ id: 'x' }] });
    expect(day.items).toEqual([]);
    expect(day.avgDurationS).toBeNull();
  });

  it('sends the details in snake case', () => {
    expect(toHotFlashDetailsBody({ severity: null, sweat: true, triggers: ['unknown'] })).toEqual({
      severity: null,
      sweat: true,
      triggers: ['unknown'],
    });
  });
});
