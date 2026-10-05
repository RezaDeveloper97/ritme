import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

import { growthSeriesSchema, measurementListSchema, measurementSchema } from './schema';

/* Boundary contract of /children/{id}/measurements + /growth against the Go goldens (B-N5-02). */
const GOLDEN_DIR = resolve(process.cwd(), '../backend-go/contract/golden/children');
interface Golden {
  steps: { request: { method: string; url: string }; status: number; body: { data?: unknown } }[];
}
const steps = (name: string, method: string, url: string): unknown[] =>
  (JSON.parse(readFileSync(resolve(GOLDEN_DIR, `${name}.json`), 'utf8')) as Golden).steps
    .filter((s) => s.request.method === method && s.request.url === url && s.status < 300)
    .map((s) => s.body.data);

describe.skipIf(!existsSync(GOLDEN_DIR))('growth parsers', () => {
  it('parses a saved measurement', () => {
    const [first] = steps('growth.fa', 'POST', '/api/v1/children/1/measurements');
    const m = measurementSchema.parse(first);
    expect(m).toMatchObject({ id: 1, source: 'measurement', measuredOn: '2026-08-12', head: null });
    expect(m.weight).toEqual({ value: 5, percentile: 42.1, inBand: true });
  });

  it('parses the history with the birth row last', () => {
    const [list] = steps('growth.fa', 'GET', '/api/v1/children/1/measurements');
    const l = measurementListSchema.parse(list);
    expect(l.measurements.length).toBeGreaterThanOrEqual(3);
    expect(l.measurements[0]?.id).toBe(2);
    expect(l.measurements.at(-1)?.source).toBe('birth');
    expect(l.verdict.status).toBe('normal');
    expect(l.disclaimer).toBeTruthy();
  });

  it('parses the weight curves and points', () => {
    const [weight] = steps('growth.fa', 'GET', '/api/v1/children/1/growth?indicator=weight');
    const s = growthSeriesSchema.parse(weight);
    expect(s).toMatchObject({ indicator: 'weight', available: true, reason: null, fromMonth: 0, toMonth: 12 });
    expect(s.reference).toHaveLength(13);
    expect(s.reference[0]).toMatchObject({ month: 0, p50: 3.232 });
    expect(s.points[0]).toMatchObject({ id: null, ageMonths: 0, value: 3.2 });
    expect(s.latest).toMatchObject({ id: 2, value: 6.1, inBand: true });
    expect(s.medianLabel).toBeTruthy();
    expect(s.bandLabel).toBeTruthy();
  });

  it('marks the curves unavailable when the sex is unknown', () => {
    const [length] = steps('growth.fa', 'GET', '/api/v1/children/1/growth?indicator=length');
    const s = growthSeriesSchema.parse(length);
    expect(s).toMatchObject({ available: false, reason: 'sex_unknown', medianLabel: null });
    expect(s.reference).toEqual([]);
    expect(s.points.every((p) => p.percentile === null)).toBe(true);
  });
});
