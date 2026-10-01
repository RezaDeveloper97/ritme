import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

import { analysisKeys } from './keys';
import { monthlyReportSchema } from './monthly';

/* Boundary contract for GET /analysis/monthly/:ym against the Go engine goldens (B-N3-07). */
const GOLDEN_DIR = resolve(process.cwd(), '../backend-go/internal/analysis/testdata/golden');
const golden = (name: string): unknown => JSON.parse(readFileSync(resolve(GOLDEN_DIR, `${name}.json`), 'utf8'));

describe.skipIf(!existsSync(GOLDEN_DIR))('monthlyReportSchema', () => {
  it('parses a full Jalali month (PDF open)', () => {
    const r = monthlyReportSchema.parse(golden('regular_monthly_j1405_06'));
    expect(r.month).toEqual({ key: '1405-06', calendar: 'jalali', from: '2026-08-23', to: '2026-09-22', complete: true });
    expect(r.headline.key).toBe('monthly.headline.busy');
    expect(r.summary.parts).toHaveLength(4);
    expect(r.metrics.map((m) => m.key)).toEqual(['cycle_length', 'days_logged', 'sleep', 'weight', 'blood_pressure', 'good_mood']);
    expect(r.metrics[4]).toEqual({
      key: 'blood_pressure',
      value: { systolic: 120, diastolic: 77 },
      delta: { systolic: 0, diastolic: 0 },
      unit: 'mmhg',
    });
    expect(r.topSymptoms).toHaveLength(3);
    expect(r.pdf).toEqual({ plus: true, locked: false });
  });

  it('parses an empty, running Gregorian month (PDF locked)', () => {
    const r = monthlyReportSchema.parse(golden('empty_monthly_2026_09'));
    expect(r.month.complete).toBe(false);
    expect(r.metrics.find((m) => m.key === 'sleep')).toMatchObject({ value: null, delta: null });
    expect(r.topSymptoms).toEqual([]);
    expect(r.pdf.locked).toBe(true);
  });

  it('drops a metric row this bundle does not know', () => {
    const raw = golden('regular_monthly_2026_09') as { metrics: unknown[] };
    const r = monthlyReportSchema.parse({ ...raw, metrics: [...raw.metrics, { key: 'steps', value: 1, delta: 0, unit: null }] });
    expect(r.metrics).toHaveLength(6);
  });

  it('keys each month by calendar', () => {
    expect(analysisKeys.monthly('1405-07', 'jalali')).toEqual(['analysis', 'monthly', 'jalali', '1405-07']);
  });
});
