import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

import { analysisKeys } from './keys';
import { bodyReportSchema, correlationsReportSchema, cycleReportSchema, periodReportSchema, symptomsReportSchema } from './reports';

/*
 * Boundary contract of the detail reports (B-N3-09) against the Go engine's
 * goldens (B-N3-07) and the contract goldens of the correlations envelope.
 */
const ROOT = resolve(process.cwd(), '../backend-go');
const GOLDEN_DIR = resolve(ROOT, 'internal/analysis/testdata/golden');
const golden = (name: string): unknown => JSON.parse(readFileSync(resolve(GOLDEN_DIR, `${name}.json`), 'utf8'));
const range = { key: '6m', from: '2026-03-26', to: '2026-09-23', days: 182 };

describe.skipIf(!existsSync(GOLDEN_DIR))('detail report parsers', () => {
  it('parses the cycle report (outlier excluded, FIGO ranges)', () => {
    const r = cycleReportSchema.parse(golden('regular_cycle_6m'));
    expect(r.cycleLength).toEqual({ median: 29, status: 'normal' });
    expect(r.figo).toEqual({ applies: true, cycleMin: 24, cycleMax: 38, periodMax: 8, variationMax: 7 });
    expect(r.variation.status).toBe('regular');
    expect(r.cycles.at(-1)).toMatchObject({ length: 38, counted: false, excluded: 'outlier' });
    expect(r.typical.days).toEqual({ period: 5, follicular: 5, fertile: 6, luteal: 13 });
    const teen = cycleReportSchema.parse(golden('teen_cycle_6m'));
    expect(teen.figo.applies).toBe(false);
    expect(teen.periodLength.status).toBe('prolonged');
    expect(cycleReportSchema.parse(golden('empty_cycle_6m')).cycles).toEqual([]);
  });

  it('parses the period report', () => {
    const r = periodReportSchema.parse(golden('regular_period_6m'));
    expect(r.peak).toEqual({ day: 2, level: 'heavy' });
    expect(r.spotting).toEqual({ days: 1, cycles: 1, ofCycles: 6 });
    expect(r.periods[0].isCurrent).toBe(true);
    expect(r.coSymptoms.items[0].pct).toBe(100);
    const empty = periodReportSchema.parse(golden('empty_period_6m'));
    expect(empty.peak).toBeNull();
    expect(empty.average).toEqual([]);
  });

  it('parses the symptoms report (moods stay out of the cycle pattern)', () => {
    const r = symptomsReportSchema.parse(golden('regular_symptoms_6m'));
    expect(r.trend.bucket).toBe('week');
    expect(r.trend.points[0].values).toHaveLength(r.trend.keys.length);
    expect(r.pattern.items.every((i) => !i.key.startsWith('mood.'))).toBe(true);
    expect(r.highlight).toMatchObject({ key: 'pain.location.head', relation: 'before_period', days: 3 });
    expect(symptomsReportSchema.parse(golden('regular_symptoms_2w')).trend.bucket).toBe('day');
    expect(symptomsReportSchema.parse(golden('empty_symptoms_6m')).highlight).toBeNull();
  });

  it('parses the correlations envelope, locked and open', () => {
    const items = (golden('regular_correlations_6m') as { items: unknown[] }).items;
    const open = correlationsReportSchema.parse({
      range,
      not_causal: true,
      plus: true,
      locked: false,
      ready: true,
      data: { days_logged: 182, items: [...items, { key: 'something_new', status: 'ready' }] },
    });
    expect(open.data?.items.map((i) => i.key)).toEqual(['sleep_mood', 'phase_energy', 'exercise_cramps', 'phase_mood']);
    expect(open.data?.items[0]).toMatchObject({ strength: 'strong', ratio: 5.3 });
    const locked = correlationsReportSchema.parse({ range, not_causal: true, plus: true, locked: true, ready: false, data: null });
    expect(locked).toMatchObject({ locked: true, ready: false, data: null });
  });

  it('parses the body report', () => {
    const r = bodyReportSchema.parse(golden('regular_body_1m'));
    expect(r.weight.current).toBe(57.8);
    expect(r.weight.movingAverageDays).toBe(7);
    expect(r.sleep.byWeekday).toHaveLength(7);
    expect(r.activity.weeks[0]).toEqual({ start: '2026-08-25', days: 7, activeDays: 3 });
    const empty = bodyReportSchema.parse(golden('empty_body_1m'));
    expect(empty.weight.ready).toBe(false);
    expect(empty.sleep.avgHours).toBeNull();
  });

  it('keys reports by name and range', () => {
    expect(analysisKeys.report('symptoms', '2w')).toEqual(['analysis', 'symptoms', '2w']);
  });
});
