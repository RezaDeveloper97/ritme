import { describe, expect, it } from 'vitest';

import { pmddChartSchema, pmddDaySchema, pmddItemsSchema } from './pmdd';

describe('pmddDaySchema', () => {
  it('keeps only 1–6 integer scores', () => {
    expect(
      pmddDaySchema.parse({ date: '2026-09-23', scores: { sadness: 5, anxiety: 0, anger: 7, focus: 2.5, mood_swings: 6 }, mean: 5.5 }),
    ).toEqual({ date: '2026-09-23', scores: { sadness: 5, mood_swings: 6 }, mean: 5.5 });
  });

  it('reads an empty day', () => {
    expect(pmddDaySchema.parse({ date: '2026-09-23', scores: {}, mean: null })).toEqual({
      date: '2026-09-23',
      scores: {},
      mean: null,
    });
  });
});

describe('pmddItemsSchema', () => {
  it('drops untitled rows and degrades to an empty list', () => {
    expect(pmddItemsSchema.parse({ items: [{ code: 'sadness', title: 'Sadness' }, { code: 'x', title: '' }] })).toEqual([
      { code: 'sadness', title: 'Sadness' },
    ]);
    expect(pmddItemsSchema.parse(null)).toEqual([]);
  });
});

describe('pmddChartSchema', () => {
  const golden = {
    required_cycles: 2,
    complete_cycles: 0,
    ready: false,
    pattern: 'not_enough_data',
    cycles: [
      {
        start: '2026-09-14',
        end: '2026-10-11',
        closed: false,
        complete: false,
        period: { from: '2026-09-14', to: '2026-09-18' },
        late_luteal: { from: '2026-10-02', to: '2026-10-11' },
        follicular: { from: '2026-09-17', to: '2026-09-23' },
        luteal_days: 0,
        follicular_days: 1,
        luteal_mean: null,
        follicular_mean: 4,
        days: [
          { date: '2026-09-23', cycle_day: 10, in_period: false, late_luteal: false, mean: 4, scores: { sadness: 4 } },
          { date: 'bad' },
        ],
      },
    ],
    alerts: [
      {
        code: 'pmdd_crisis',
        title: 'If you think about hurting yourself',
        body: 'Call now.',
        meta: {
          severity: 'urgent',
          hotlines: [
            { number: '1480', label: 'Counselling line' },
            { number: 'javascript:alert(1)', label: 'bad' },
          ],
        },
        audiences: null,
        needs_review: true,
      },
    ],
  };

  it('parses the golden shape and drops malformed rows', () => {
    const chart = pmddChartSchema.parse(golden);
    expect(chart.pattern).toBe('not_enough_data');
    expect(chart.cycles[0].lateLuteal).toEqual({ from: '2026-10-02', to: '2026-10-11' });
    expect(chart.cycles[0].days).toEqual([
      { date: '2026-09-23', cycleDay: 10, inPeriod: false, lateLuteal: false, mean: 4 },
    ]);
    expect(chart.alerts).toEqual([
      {
        code: 'pmdd_crisis',
        title: 'If you think about hurting yourself',
        body: 'Call now.',
        severity: 'urgent',
        hotlines: [{ number: '1480', label: 'Counselling line' }],
      },
    ]);
  });

  it('falls back on an unknown pattern', () => {
    expect(pmddChartSchema.parse({ ...golden, pattern: 'other' }).pattern).toBe('not_enough_data');
  });
});
