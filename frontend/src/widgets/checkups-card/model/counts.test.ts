import { describe, expect, it } from 'vitest';

import type { CheckupItem } from '@/entities/checkup';

import { BOOK_HREF, bookPrefill, countParts, countsLine, highlightKind, highlightMeta, ringFraction } from './counts';

const item = (over: Partial<CheckupItem>): CheckupItem => ({
  id: 3,
  key: 'pap_smear',
  title: 'Pap',
  subtitle: null,
  category: 'multi_year',
  section: 'overdue',
  status: 'overdue',
  icon: 'shield',
  tone: 'violet',
  intervalLabel: 'هر ۳ سال',
  timingLabel: 'روز ۱۰ تا ۲۰ سیکل',
  lastDoneOn: '2022-04-10',
  nextDueOn: '2025-04-10',
  nextDueLabel: 'عقب‌افتاده از فروردین',
  isCustom: false,
  ...over,
});

describe('highlight rows (v14_Main)', () => {
  it('an overdue cycle-timed Pap is a «ثبت نوبت» booking row', () => {
    expect(highlightKind(item({}))).toBe('book');
  });

  it('the self-exam stays a «راهنما» row, due or overdue', () => {
    expect(highlightKind(item({ key: 'breast_self_exam', status: 'overdue' }))).toBe('guide');
    expect(highlightKind(item({ key: 'breast_self_exam', status: 'due' }))).toBe('guide');
  });

  it('the sub-line says when: overdue-since alone, or days left + the cycle window', () => {
    expect(highlightMeta(item({}), '، ')).toBe('عقب‌افتاده از فروردین');
    expect(highlightMeta(item({ status: 'due', nextDueLabel: '۳ روز دیگر', timingLabel: 'روز ۷ تا ۱۰ سیکل' }), '، ')).toBe(
      '۳ روز دیگر، روز ۷ تا ۱۰ سیکل',
    );
    expect(highlightMeta(item({ status: 'due', nextDueLabel: null, timingLabel: null, subtitle: 'x' }), '، ')).toBe('x');
  });
});

const label = (key: string, n: number) => `${n} ${key}`;

describe('countsLine', () => {
  it('joins all three non-zero parts in order', () => {
    expect(countsLine({ total: 6, upToDate: 4, due: 1, overdue: 1 }, label, ' · ')).toBe(
      '4 countUpToDate · 1 countDue · 1 countOverdue',
    );
  });

  it('omits zero parts', () => {
    expect(countsLine({ total: 6, upToDate: 4, due: 0, overdue: 2 }, label, ' · ')).toBe(
      '4 countUpToDate · 2 countOverdue',
    );
    expect(countParts({ total: 2, upToDate: 0, due: 2, overdue: 0 })).toEqual([
      { key: 'countDue', count: 2 },
    ]);
  });

  it('is empty when every count is zero', () => {
    expect(countsLine({ total: 0, upToDate: 0, due: 0, overdue: 0 }, label, ' · ')).toBe('');
  });
});

describe('ringFraction', () => {
  it('clamps and handles zero total', () => {
    expect(ringFraction({ total: 0, upToDate: 0, due: 0, overdue: 0 })).toBe(0);
    expect(ringFraction({ total: 4, upToDate: 2, due: 2, overdue: 0 })).toBe(0.5);
  });
});

describe('book an overdue checkup (audit M3-M7 #3)', () => {
  it('keeps the checkup title out of the URL', () => {
    const url = new URL(BOOK_HREF, 'https://x');
    expect(url.pathname).toBe('/reminders/appointment/new');
    expect([...url.searchParams.keys()]).toEqual(['kind']);
    expect(bookPrefill({ title: 'آزمایش تیروئید من' })).toEqual({ title: 'آزمایش تیروئید من' });
  });
});
