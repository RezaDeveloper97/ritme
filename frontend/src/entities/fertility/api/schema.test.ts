import { describe, expect, it } from 'vitest';

import { fertilityKeys } from './keys';
import {
  fertilityBbtSchema,
  fertilityDaySchema,
  fertilityInsightsSchema,
  fertilityTodaySchema,
} from './schema';

/*
 * Boundary contract for `/api/v1/fertility/*` — fixtures follow
 * docs/fertility-ttc/README.md and the v19 artboards. The frontend is written
 * before the Go endpoints (T-M5-01…03), so the parsers must also survive
 * values this bundle has never seen and the shapes the README leaves open.
 */

describe('fertilityTodaySchema', () => {
  const today = {
    date: '2026-09-20',
    cycle_day: 12,
    chance: { level: 'high', label: 'زیاد', bars: 4 },
    lh: { value: null, label: 'ثبت نشده' },
    bbt: { value: 36.42 },
    intercourse: { value: 'unprotected', label: 'بدون محافظت' },
  };

  it('maps the README shape', () => {
    expect(fertilityTodaySchema.parse(today)).toEqual({
      date: '2026-09-20',
      cycleDay: 12,
      chance: { level: 'high', label: 'زیاد', bars: 4 },
      lh: { value: null, label: 'ثبت نشده' },
      bbt: 36.42,
      intercourse: { value: 'unprotected', label: 'بدون محافظت' },
    });
  });

  it('reads bare enums, decimal strings and a bare bbt', () => {
    const parsed = fertilityTodaySchema.parse({
      ...today,
      lh: 'positive',
      bbt: '36.40',
      intercourse: 'protected',
    });
    expect(parsed.lh).toEqual({ value: 'positive', label: null });
    expect(parsed.bbt).toBe(36.4);
    expect(parsed.intercourse).toEqual({ value: 'protected', label: null });
  });

  it('turns unknown enums into null and clamps the bars', () => {
    const parsed = fertilityTodaySchema.parse({
      ...today,
      chance: { level: 'astronomical', label: 'x', bars: 9 },
      lh: { value: 'blinking', label: 'چشمک' },
      intercourse: { value: 'unknown_kind', label: null },
    });
    expect(parsed.chance).toEqual({ level: null, label: 'x', bars: 5 });
    expect(parsed.lh).toEqual({ value: null, label: 'چشمک' });
    expect(parsed.intercourse).toEqual({ value: null, label: null });
  });

  it('defaults every missing block', () => {
    expect(fertilityTodaySchema.parse({ date: '2026-09-20T00:00:00Z' })).toEqual({
      date: '2026-09-20',
      cycleDay: null,
      chance: { level: null, label: null, bars: 0 },
      lh: { value: null, label: null },
      bbt: null,
      intercourse: { value: null, label: null },
    });
  });

  it('drops negative / junk bar counts and bbt', () => {
    const parsed = fertilityTodaySchema.parse({ ...today, chance: { level: 'low', bars: -2 }, bbt: 'n/a' });
    expect(parsed.chance.bars).toBe(0);
    expect(parsed.bbt).toBeNull();
  });
});

describe('fertilityDaySchema', () => {
  const day = {
    date: '2026-09-20',
    cycle_day: 12,
    lh: 'positive',
    mucus: 'egg_white',
    bbt: 36.42,
    bbt_time: '06:45:00',
    intercourse: 'unprotected',
    symptoms: ['ovarian_pain'],
    note: 'یادداشت',
    chance: { level: 'high', label: 'زیاد', bars: 4 },
  };

  it('maps the merged day', () => {
    expect(fertilityDaySchema.parse(day)).toEqual({
      date: '2026-09-20',
      cycleDay: 12,
      lh: 'positive',
      mucus: 'egg_white',
      bbt: 36.42,
      bbtTime: '06:45',
      intercourse: 'unprotected',
      symptoms: ['ovarian_pain'],
      note: 'یادداشت',
      chance: { level: 'high', label: 'زیاد', bars: 4 },
    });
  });

  it('parses an empty day to all-null', () => {
    expect(fertilityDaySchema.parse({ date: '2026-09-20' })).toEqual({
      date: '2026-09-20',
      cycleDay: null,
      lh: null,
      mucus: null,
      bbt: null,
      bbtTime: null,
      intercourse: null,
      symptoms: [],
      note: null,
      chance: null,
    });
  });

  it('nulls unknown enums, drops unknown symptoms and orders the rest', () => {
    const parsed = fertilityDaySchema.parse({
      ...day,
      lh: 'glowing',
      mucus: 'watery',
      intercourse: 'other',
      symptoms: ['spotting', 'headache', 'bloating', 'spotting', { value: 'breast_sensitivity' }],
      note: '   ',
      chance: 'high',
    });
    expect(parsed.lh).toBeNull();
    expect(parsed.mucus).toBeNull();
    expect(parsed.intercourse).toBeNull();
    expect(parsed.symptoms).toEqual(['bloating', 'breast_sensitivity', 'spotting']);
    expect(parsed.note).toBeNull();
    expect(parsed.chance).toBeNull();
  });

  it('accepts {value,label} enums on the day too', () => {
    const parsed = fertilityDaySchema.parse({ ...day, lh: { value: 'faint', label: 'کم‌رنگ' } });
    expect(parsed.lh).toBe('faint');
  });

  it('requires a date', () => {
    expect(() => fertilityDaySchema.parse({ lh: 'positive' })).toThrow();
  });
});

describe('fertilityBbtSchema', () => {
  const cycle = {
    start_date: '2026-09-09',
    points: [
      { cycle_day: 2, date: '2026-09-10', value: '36.30' },
      { cycle_day: 1, date: '2026-09-09', value: 36.25 },
      { cycle_day: 3, date: '2026-09-11', value: null },
      { cycle_day: 'x', date: '2026-09-12', value: 36.4 },
    ],
    coverline: 36.4,
    fertile_window: { from_day: 10, to_day: 15 },
    shift_day: null,
    phase: 'pre_shift',
  };

  it('reads {cycles:[…]} with top-level stats and tip', () => {
    const parsed = fertilityBbtSchema.parse({
      range: 3,
      cycles: [cycle, { ...cycle, start_date: '2026-08-11', shift_day: 16, phase: 'post_shift' }],
      stats: { pre_ovulation_avg: 36.32, logged_days: 12, cycle_days_so_far: 12, gaps: 0 },
      past_shift_days: [15, '16'],
      tip: { title: 'دنبال جهش ۰٫۲ تا ۰٫۵ درجه باش', body: 'اگر دما سه روز…' },
    });
    expect(parsed.range).toBe(3);
    expect(parsed.cycles).toHaveLength(2);
    expect(parsed.cycles[0]).toEqual({
      startDate: '2026-09-09',
      points: [
        { cycleDay: 1, date: '2026-09-09', value: 36.25 },
        { cycleDay: 2, date: '2026-09-10', value: 36.3 },
      ],
      coverline: 36.4,
      fertileWindow: { fromDay: 10, toDay: 15 },
      shiftDay: null,
      phase: 'pre_shift',
    });
    expect(parsed.cycles[1]?.phase).toBe('post_shift');
    expect(parsed.stats).toEqual({ preOvulationAvg: 36.32, loggedDays: 12, cycleDaysSoFar: 12, gaps: 0 });
    expect(parsed.pastShiftDays).toEqual([15, 16]);
    expect(parsed.tip).toEqual({ title: 'دنبال جهش ۰٫۲ تا ۰٫۵ درجه باش', body: 'اگر دما سه روز…' });
  });

  it('reads a single cycle at the top level with in-cycle stats and a string tip', () => {
    const parsed = fertilityBbtSchema.parse({
      ...cycle,
      phase: 'mystery',
      fertile_window: { from_day: 15, to_day: 10 },
      stats: { logged_days: '5' },
      tip: 'یک نکته',
    });
    expect(parsed.range).toBe(1);
    expect(parsed.cycles).toHaveLength(1);
    expect(parsed.cycles[0]?.phase).toBeNull();
    expect(parsed.cycles[0]?.fertileWindow).toBeNull();
    expect(parsed.stats).toEqual({ preOvulationAvg: null, loggedDays: 5, cycleDaysSoFar: 0, gaps: 0 });
    expect(parsed.tip).toEqual({ title: null, body: 'یک نکته' });
  });

  it('survives an empty / junk payload', () => {
    expect(fertilityBbtSchema.parse(null)).toEqual({
      range: 1,
      cycles: [],
      stats: { preOvulationAvg: null, loggedDays: 0, cycleDaysSoFar: 0, gaps: 0 },
      pastShiftDays: [],
      tip: null,
    });
    expect(fertilityBbtSchema.parse({ range: 4, cycles: 'nope' }).range).toBe(1);
  });
});

describe('fertilityInsightsSchema', () => {
  it('maps the README shape', () => {
    expect(
      fertilityInsightsSchema.parse({
        cycles_used: 6,
        window: { start: '2026-09-19', end: '2026-09-24', ovulation: '2026-09-23' },
        confidence: 'medium',
        evidence: [
          { key: 'cycles', title: '۶ سیکل کامل ثبت شده', detail: 'طول معمول ۲۹ روز، نوسان ±۲', strength: 'strong' },
          { key: 'bbt_shift', title: 'جهش دما در ۲ سیکل قبل', detail: 'روز ۱۵ و ۱۶ سیکل', strength: 'medium' },
          { key: 'lh', title: 'تست LH', detail: 'هنوز در این سیکل ثبت نشده', strength: 'none' },
        ],
        history: [
          { month_label: 'شهریور', ovulation_day: 16 },
          { month_label: 'مرداد', ovulation_day: '15' },
        ],
        tips: ['دمای پایه را هر صبح ثبت کن', { title: 'LH', body: 'از روز ۱۰ تست بزن' }],
      }),
    ).toEqual({
      cyclesUsed: 6,
      window: { start: '2026-09-19', end: '2026-09-24', ovulation: '2026-09-23' },
      confidence: 'medium',
      evidence: [
        { key: 'cycles', title: '۶ سیکل کامل ثبت شده', detail: 'طول معمول ۲۹ روز، نوسان ±۲', strength: 'strong' },
        { key: 'bbt_shift', title: 'جهش دما در ۲ سیکل قبل', detail: 'روز ۱۵ و ۱۶ سیکل', strength: 'medium' },
        { key: 'lh', title: 'تست LH', detail: 'هنوز در این سیکل ثبت نشده', strength: 'none' },
      ],
      history: [
        { monthLabel: 'شهریور', ovulationDay: 16 },
        { monthLabel: 'مرداد', ovulationDay: 15 },
      ],
      tips: ['دمای پایه را هر صبح ثبت کن', 'از روز ۱۰ تست بزن'],
    });
  });

  it('nulls unknown enums, drops broken rows and a half window', () => {
    const parsed = fertilityInsightsSchema.parse({
      cycles_used: 'many',
      window: { start: '2026-09-19' },
      confidence: 'certain',
      evidence: [{ key: 'x', title: 'ok', strength: 'overwhelming' }, { key: 'y' }],
      history: [{ month_label: 'تیر', ovulation_day: null }, { ovulation_day: 4 }],
      tips: 'not a list',
    });
    expect(parsed).toEqual({
      cyclesUsed: 0,
      window: null,
      confidence: null,
      evidence: [{ key: 'x', title: 'ok', detail: null, strength: null }],
      history: [{ monthLabel: 'تیر', ovulationDay: null }],
      tips: [],
    });
  });

  it('parses an empty object (new user, no cycles yet)', () => {
    expect(fertilityInsightsSchema.parse({})).toEqual({
      cyclesUsed: 0,
      window: null,
      confidence: null,
      evidence: [],
      history: [],
      tips: [],
    });
  });
});

describe('fertilityKeys', () => {
  it('nests every read under `all` so one invalidation refreshes them', () => {
    const all = fertilityKeys.all;
    for (const key of [
      fertilityKeys.today(),
      fertilityKeys.day('2026-09-20'),
      fertilityKeys.bbt(3),
      fertilityKeys.insights(),
    ]) {
      expect(key.slice(0, all.length)).toEqual([...all]);
    }
    expect(fertilityKeys.day('2026-09-20').slice(0, 2)).toEqual([...fertilityKeys.daysAll()]);
    expect(fertilityKeys.bbt()).toEqual([...fertilityKeys.bbtAll(), 1]);
  });
});
