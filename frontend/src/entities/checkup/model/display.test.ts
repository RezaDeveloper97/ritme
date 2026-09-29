import { describe, expect, it } from 'vitest';

import { checkupResultIcon, checkupStatusIcon, formatCheckupMonth } from './display';

describe('checkup display helpers', () => {
  it('status and result glyphs follow v14', () => {
    expect(checkupStatusIcon('due')).toBe('bellPlain');
    expect(checkupStatusIcon('overdue')).toBe('info');
    expect(checkupStatusIcon('soon')).toBe('clock');
    expect(checkupStatusIcon('up_to_date')).toBe('check');
    expect(checkupResultIcon('normal')).toBe('check');
    expect(checkupResultIcon('follow_up')).toBe('info');
    expect(checkupResultIcon('pending')).toBe('clock');
  });

  it('month + year in the locale calendar', () => {
    expect(formatCheckupMonth('2025-04-10', 'fa')).toBe('فروردین ۱۴۰۴');
    expect(formatCheckupMonth('2025-04-10', 'en')).toBe('April 2025');
  });
});
