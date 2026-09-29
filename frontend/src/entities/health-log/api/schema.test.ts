import { describe, expect, it } from 'vitest';

import { pickDayLog } from './schema';

describe('pickDayLog', () => {
  it('reads an empty one-day page as no log (no 404 for an unlogged day)', () => {
    expect(pickDayLog({ current_page: 1, data: [], total: 0 })).toBeNull();
  });

  it('returns the day row, dropping empty fields', () => {
    const log = pickDayLog({
      current_page: 1,
      data: [{ log_date: '2026-09-29', bleeding_intensity: 'light', notes: null }],
      total: 1,
    });
    expect(log).toEqual({ log_date: '2026-09-29', bleeding_intensity: 'light' });
  });

  it('rejects a body that is not a paginator', () => {
    expect(() => pickDayLog({ log_date: '2026-09-29' })).toThrow();
  });
});
