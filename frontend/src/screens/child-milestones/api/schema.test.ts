import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

import { chipUnit } from '../model/months';
import { milestonesSchema } from './schema';

/* Boundary contract of /children/{id}/milestones against the Go goldens (B-N5-02). */
const GOLDEN_DIR = resolve(process.cwd(), '../backend-go/contract/golden/children');
interface Golden {
  steps: { request: { method: string; url: string }; status: number; body: { data?: unknown } }[];
}
const step = (method: string, url: string): unknown =>
  (JSON.parse(readFileSync(resolve(GOLDEN_DIR, 'milestones_and_learn.fa.json'), 'utf8')) as Golden).steps.find(
    (s) => s.request.method === method && s.request.url === url,
  )?.body.data;

describe.skipIf(!existsSync(GOLDEN_DIR))('milestones parser', () => {
  it('parses the current band', () => {
    const v = milestonesSchema.parse(step('GET', '/api/v1/children/1/milestones'));
    expect(v.bands.find((b) => b.current)?.months).toBe(3);
    expect(v.band).toMatchObject({ months: 3, checked: 0, total: 7 });
    expect(v.band.items[0]).toMatchObject({ code: 'm3_social_smile', checked: false, checkedOn: null });
    expect(v.band.activities).toHaveLength(2);
    expect(v.band.doctorNote).toBeTruthy();
    expect(v.intro).toBeTruthy();
  });

  it('parses a check and another month', () => {
    const put = milestonesSchema.parse(step('PUT', '/api/v1/children/1/milestones/m3_social_smile'));
    expect(put.band.checked).toBe(1);
    expect(put.band.items[0]).toMatchObject({ checked: true, checkedOn: '2026-09-23' });
    const six = milestonesSchema.parse(step('GET', '/api/v1/children/1/milestones?month=6'));
    expect(six.band.months).toBe(6);
  });
});

describe('month chips', () => {
  it('shows whole years from two years on', () => {
    expect(chipUnit(3)).toEqual({ unit: 'months', n: 3 });
    expect(chipUnit(18)).toEqual({ unit: 'months', n: 18 });
    expect(chipUnit(24)).toEqual({ unit: 'years', n: 2 });
    expect(chipUnit(30)).toEqual({ unit: 'months', n: 30 });
  });
});
