import { describe, expect, it } from 'vitest';

import type { CheckupItem } from '@/entities/checkup';

import { barSegments, filterHref, groupBySection, nextFilter, parseFilter, rowMeta, worstStatus } from './view';

const item = (id: number, section: CheckupItem['section']): CheckupItem => ({
  id,
  key: null,
  title: `t${id}`,
  subtitle: null,
  category: 'annual',
  section,
  status: 'up_to_date',
  icon: null,
  tone: 'neutral',
  intervalLabel: null,
  timingLabel: null,
  lastDoneOn: null,
  nextDueOn: null,
  nextDueLabel: null,
  isCustom: false,
});

describe('checkups view', () => {
  it('parses and builds filter urls', () => {
    expect(parseFilter('action')).toBe('action');
    expect(parseFilter('x')).toBe('all');
    expect(parseFilter(undefined)).toBe('all');
    expect(filterHref('all')).toBe('/checkups');
    expect(filterHref('done')).toBe('/checkups?filter=done');
  });

  it('moves across tabs with RTL-aware arrows', () => {
    expect(nextFilter('all', 'ArrowLeft', 'rtl')).toBe('action');
    expect(nextFilter('all', 'ArrowLeft', 'ltr')).toBe('done');
    expect(nextFilter('all', 'End', 'ltr')).toBe('done');
    expect(nextFilter('all', 'a', 'ltr')).toBeNull();
  });

  it('groups by section in server order', () => {
    const groups = groupBySection([item(1, 'overdue'), item(2, 'annual'), item(3, 'overdue')]);
    expect(groups.map((g) => g.section)).toEqual(['overdue', 'annual']);
    expect(groups[0].items.map((i) => i.id)).toEqual([1, 3]);
  });

  it('puts this month first and overdue second, whatever the server order', () => {
    const groups = groupBySection([
      item(1, 'annual'),
      item(2, 'overdue'),
      item(3, 'age_based'),
      item(4, 'this_month'),
      item(5, 'six_monthly'),
      item(6, 'overdue'),
    ]);
    expect(groups.map((g) => g.section)).toEqual(['this_month', 'overdue', 'annual', 'age_based', 'six_monthly']);
    expect(groups[1].items.map((i) => i.id)).toEqual([2, 6]);
  });

  it('puts overdue first when there is nothing this month', () => {
    const groups = groupBySection([item(1, 'annual'), item(2, 'monthly'), item(3, 'overdue')]);
    expect(groups.map((g) => g.section)).toEqual(['overdue', 'annual', 'monthly']);
  });

  it('picks the worst status and builds bar segments', () => {
    expect(worstStatus({ total: 3, upToDate: 3, due: 0, overdue: 0 })).toBe('up_to_date');
    expect(worstStatus({ total: 3, upToDate: 1, due: 2, overdue: 0 })).toBe('due');
    expect(worstStatus({ total: 3, upToDate: 1, due: 1, overdue: 1 })).toBe('overdue');
    expect(barSegments({ total: 0, upToDate: 0, due: 0, overdue: 0 })).toEqual([]);
    const segs = barSegments({ total: 4, upToDate: 2, due: 0, overdue: 2 });
    expect(segs.map((s) => [s.status, s.percent])).toEqual([
      ['up_to_date', 50],
      ['overdue', 50],
    ]);
  });
});

describe('rowMeta', () => {
  it('interval + cycle window, else interval + subtitle', () => {
    expect(rowMeta({ intervalLabel: 'هر ماه', timingLabel: 'روز ۷ تا ۱۰ سیکل', subtitle: 'x' }, '، ')).toBe(
      'هر ماه، روز ۷ تا ۱۰ سیکل',
    );
    expect(rowMeta({ intervalLabel: 'هر سال', timingLabel: null, subtitle: 'توسط پزشک' }, '، ')).toBe('هر سال، توسط پزشک');
    expect(rowMeta({ intervalLabel: null, timingLabel: null, subtitle: null }, '، ')).toBe('');
  });
});
