import { describe, expect, it } from 'vitest';

import { cycleOverviewSchema, type CycleTodayUi } from '@/entities/cycle';

import { enginePredictionRows } from './prediction-card';

const ui = (card: Partial<NonNullable<CycleTodayUi['predictionCard']>> | null): CycleTodayUi => ({
  ringAction: null,
  banner: null,
  showPregnancyTestButton: false,
  predictionCard: card && {
    period: { start: '2026-08-01', end: '2026-08-05' },
    pms: { start: '2026-07-28', end: '2026-07-31' },
    fertile: { start: '2026-07-12', end: '2026-07-17' },
    ovulation: { date: '2026-07-17', source: 'calendar' },
    cycleLength: 28,
    cycleRegularity: 'regular',
    ...card,
  },
});

describe('enginePredictionRows', () => {
  it('is null without a v6 card (Laravel path unchanged)', () => {
    expect(enginePredictionRows(null)).toBeNull();
    expect(enginePredictionRows(ui(null))).toBeNull();
  });

  it.each(['bbt_confirmed', 'lh_positive', 'calendar'] as const)('keeps the ovulation source %s', (source) => {
    const rows = enginePredictionRows(ui({ ovulation: { date: '2026-07-17', source } }));
    expect(rows?.ovulation).toEqual({ date: '2026-07-17', source });
  });

  it('has no ovulation row without a date', () => {
    expect(enginePredictionRows(ui({ ovulation: { date: null, source: 'calendar' } }))?.ovulation).toBeNull();
  });

  it('hides the fertile window when the engine sends null', () => {
    const rows = enginePredictionRows(ui({ fertile: null }));
    expect(rows?.fertile).toBeNull();
    expect(rows?.period).toEqual({ start: '2026-08-01', end: '2026-08-05' });
  });

  it.each(['regular', 'irregular', 'insufficient_data'] as const)('passes regularity %s', (r) => {
    expect(enginePredictionRows(ui({ cycleRegularity: r }))?.regularity).toBe(r);
  });
});

describe('overview regularity (both enums)', () => {
  const parse = (regularity: string) =>
    cycleOverviewSchema.parse({
      date: '2026-07-24',
      logging: { streak_days: 0, logged_today: false },
      next_period: null,
      pms_window: null,
      cycle_length: { median: 28, source: 'x', based_on_cycles: 3, spread_days: 1, regularity },
    }).cycleLength.regularity;

  it.each([
    ['regular', 'regular'],
    ['irregular', 'irregular'],
    ['insufficient_data', 'insufficient_data'],
    ['relatively_regular', 'regular'],
    ['irregular_possible', 'irregular'],
    ['not_enough_data', 'insufficient_data'],
    ['something_new', null],
  ])('%s -> %s', (raw, out) => {
    expect(parse(raw)).toBe(out);
  });
});
