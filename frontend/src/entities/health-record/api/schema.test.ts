import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

import { addAllergy, removeAllergy, toggleCode } from '../model/edit';
import { GYN_CONDITIONS } from '../model/types';
import { healthRecordSchema, pregnancyEntrySchema } from './schema';

/* Boundary contract of /api/v1/health-record* against the Go goldens (B-N6-03). */
const GOLDEN_DIR = resolve(process.cwd(), '../backend-go/contract/golden/healthrecord');
interface Step {
  request: { method: string; url: string };
  status: number;
  body: { data?: unknown };
}
const steps = (name: string): Step[] =>
  (JSON.parse(readFileSync(resolve(GOLDEN_DIR, `${name}.json`), 'utf8')) as { steps: Step[] }).steps;

describe('healthRecordSchema', () => {
  it('parses the recorded record of a user with a profile and cycles', () => {
    const r = healthRecordSchema.parse(steps('show.fa')[0]?.body.data);
    expect(r.basics.data.heightCm).toBe(165);
    expect(r.basics.data.bmi?.value).toBe(22.2);
    expect(r.basics.editable).toBe(true);
    expect(r.cycle.data.medianCycle).toBe(28);
    expect(r.cycle.data.topSymptoms[0]?.label).toBe('گرفتگی');
    expect(r.vitals.empty).toBe(true);
    expect(r.labs.data.items).toEqual([]);
  });

  it('parses the empty record', () => {
    const r = healthRecordSchema.parse(steps('show_no_profile')[0]?.body.data);
    expect(r.basics.empty).toBe(true);
    expect(r.allergies.data.items).toBeNull();
    expect(r.pregnancies.data.pregnanciesCount).toBe(0);
  });

  it('parses the manual pregnancy entries and the record after edits', () => {
    const flow = steps('pregnancies_flow');
    const created = pregnancyEntrySchema.parse(flow[1]?.body.data);
    expect(created).toMatchObject({ source: 'manual', outcome: 'vaginal', babyCount: 1, editable: true });
    const rec = flow.find((s) => s.request.method === 'GET');
    const r = healthRecordSchema.parse(rec?.body.data);
    expect(r.pregnancies.data.items.map((i) => i.outcome)).toEqual(['cesarean', 'ended']);
    const basics = healthRecordSchema.parse(steps('basics_flow')[1]?.body.data);
    expect(basics.basics.data.bloodType).toBe('A+');
    expect(basics.allergies.data.items).toEqual(['Penicillin', 'Pollen']);
  });
});

describe('edit helpers', () => {
  it('adds allergies once, trimmed', () => {
    expect(addAllergy(['Penicillin'], '  penicillin ')).toEqual(['Penicillin']);
    expect(addAllergy(['Penicillin'], ' pollen   dust ')).toEqual(['Penicillin', 'pollen dust']);
    expect(addAllergy([], '   ')).toEqual([]);
    expect(removeAllergy(['a', 'b'], 'a')).toEqual(['b']);
  });

  it('toggles codes in canonical order', () => {
    expect(toggleCode(['fibroids'], 'pcos', GYN_CONDITIONS)).toEqual(['pcos', 'fibroids']);
    expect(toggleCode(['pcos', 'fibroids'], 'pcos', GYN_CONDITIONS)).toEqual(['fibroids']);
  });
});
