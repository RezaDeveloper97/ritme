import { describe, expect, it } from 'vitest';

import {
  checkupPillClass,
  checkupPillIcon,
  checkupPillKind,
  checkupResultIcon,
  checkupStatusIcon,
  formatCheckupMonth,
} from './display';

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

  it('v6 pills: follow-up amber, informational and after-pregnancy calm', () => {
    expect(checkupPillKind({ status: 'overdue', label: 'after_pregnancy' })).toBe('after_pregnancy');
    expect(checkupPillKind({ status: 'disabled', label: 'after_pregnancy' })).toBe('disabled');
    expect(checkupPillKind({ status: 'follow_up', label: null })).toBe('follow_up');
    expect(checkupPillClass('follow_up')).toBe('ck-result-follow_up');
    expect(checkupPillClass('informational')).toBe('ck-status-not_yet');
    expect(checkupPillClass('after_pregnancy')).toBe('ck-status-not_yet');
    expect(checkupPillIcon('after_pregnancy')).toBe('mother');
    expect(checkupPillIcon('informational')).toBe('eye');
  });
});
