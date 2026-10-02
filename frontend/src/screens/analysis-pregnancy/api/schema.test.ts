import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

import { pregnancyAnalysisSchema } from './schema';

/* Boundary contract of GET /analysis/pregnancy against the Go engine goldens (B-N3-12). */
const GOLDEN_DIR = resolve(process.cwd(), '../backend-go/internal/analysis/testdata/golden');
const golden = (name: string): unknown => JSON.parse(readFileSync(resolve(GOLDEN_DIR, `${name}.json`), 'utf8'));

describe.skipIf(!existsSync(GOLDEN_DIR))('pregnancy analysis parser', () => {
  it('parses a full pregnancy (Plus open)', () => {
    const r = pregnancyAnalysisSchema.parse(golden('preg_full_pregnancy'));
    expect(r.pregnancy).toMatchObject({ week: 31, trimester: 3, dueDate: '2026-11-27' });
    const w = r.sections.weightGain.data;
    expect(w?.status).toBe('within');
    expect(w?.bmiCategory).toBe('normal');
    expect(w?.baseline?.source).toBe('before_pregnancy');
    expect(w?.band).toHaveLength(3);
    expect(w?.iomTable.map((x) => x.category)).toEqual(['underweight', 'normal', 'overweight', 'obese']);
    expect(w?.current?.recommended).not.toBeNull();
    expect(r.sections.bloodPressure.data?.status).toBe('below_threshold');
    expect(r.sections.glucose.data?.slots.map((s) => s.slot)).toEqual(['fasting', 'one_hour', 'two_hour']);
    expect(r.sections.kicks.data?.days).toHaveLength(7);
    expect(r.sections.kicks.data?.allWithinWindow).toBe(false);
    expect(r.sections.symptoms.data?.trimesters[0].items[0].label).toBe('تهوع');
    expect(r.sections.visits.data?.next?.title).toBe('سونوگرافی');
  });

  it('parses locked Plus sections and missing data', () => {
    const sparse = pregnancyAnalysisSchema.parse(golden('preg_sparse_pregnancy'));
    expect(sparse.sections.glucose).toMatchObject({ plus: true, locked: true, data: null });
    expect(sparse.sections.symptoms.locked).toBe(true);
    expect(sparse.sections.weightGain.data?.missing).toBe('height');
    expect(sparse.sections.bloodPressure.data?.status).toBe('high');
    const early = pregnancyAnalysisSchema.parse(golden('preg_early_pregnancy'));
    expect(early.sections.weightGain.data?.missing).toBe('baseline');
    expect(early.sections.weightGain.data?.points).toEqual([]);
    expect(early.sections.visits.ready).toBe(false);
  });

  it('degrades an unknown enum to null', () => {
    const raw = golden('preg_full_pregnancy') as { sections: { weight_gain: { data: { status: string } } } };
    raw.sections.weight_gain.data.status = 'way_off';
    expect(pregnancyAnalysisSchema.parse(raw).sections.weightGain.data?.status).toBeNull();
  });
});
