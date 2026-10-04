import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

import { toChildBody } from './queries';
import { childHomeSchema, childrenListSchema, childSchema, parseCompanionChildren } from './schema';

/* Boundary contract of /api/v1/children* against the Go contract goldens (B-N5-02). */
const GOLDEN_DIR = resolve(process.cwd(), '../backend-go/contract/golden/children');
interface Golden {
  steps: { request: { method: string; url: string }; status: number; body: { data?: unknown } }[];
}
const golden = (name: string): Golden =>
  JSON.parse(readFileSync(resolve(GOLDEN_DIR, `${name}.json`), 'utf8')) as Golden;
const stepData = (name: string, method: string, url: string): unknown =>
  golden(name).steps.find((s) => s.request.method === method && s.request.url === url)?.body.data;

describe.skipIf(!existsSync(GOLDEN_DIR))('children parsers', () => {
  it('parses a created child card', () => {
    const c = childSchema.parse(stepData('child_home.fa', 'POST', '/api/v1/children'));
    expect(c).toMatchObject({ id: 1, name: 'آوا', initial: 'آ', sex: 'girl', role: 'owner', canEdit: true, ownerName: null });
    expect(c.age).toMatchObject({ months: 3, weeks: 14, daysInMonth: 11 });
    expect(c.birth).toEqual({ weightKg: 3.2, lengthCm: 50, headCm: null });
    expect(c.vaccines.next).toMatchObject({ code: 'birth', status: 'overdue', total: 3 });
    expect(c.vaccines.next?.doseNames).toHaveLength(3);
    expect(c.growth.status).toBe('normal');
  });

  it('parses the list with its counters', () => {
    const l = childrenListSchema.parse(stepData('child_home.fa', 'GET', '/api/v1/children'));
    expect(l).toMatchObject({ count: 1, ownedCount: 1, maxChildren: 10, canAdd: true });
    expect(l.sharingNote).toBeTruthy();
    const empty = childrenListSchema.parse(stepData('index_empty.en', 'GET', '/api/v1/children'));
    expect(empty.children).toEqual([]);
  });

  it('parses the child home', () => {
    const h = childHomeSchema.parse(stepData('child_home.fa', 'GET', '/api/v1/children/1'));
    expect(h.latest).toMatchObject({ source: 'birth', head: null });
    expect(h.latest?.weight).toMatchObject({ value: 3.2, inBand: true });
    expect(h.milestones).toEqual({ bandMonths: 3, label: '۳ ماهگی', checked: 0, total: 7 });
    expect(h.thisWeek?.weeks).toBe(14);
    expect(h.learn.featured?.code).toBe('night_sleep_3m');
    // `today` is null until B-N5-03 lands its baby-log summary; the screen keeps «به‌زودی» either way.
    expect(h.today === null || typeof h.today === 'object').toBe(true);
  });

  it('marks a spouse-shared child read-only and parses the companion card', () => {
    const g = golden('shared_with_spouse');
    const list = g.steps.find((s) => s.request.url === '/api/v1/children' && s.request.method === 'GET');
    const l = childrenListSchema.parse(list?.body.data);
    expect(l.children[0]).toMatchObject({ role: 'shared', canEdit: false, ownerName: 'Contract regular' });
    const home = g.steps.find((s) => s.request.url === '/api/v1/companion/home')?.body.data as {
      partners: { child: unknown }[];
    };
    const kids = parseCompanionChildren(home.partners[0].child);
    expect(kids).toHaveLength(1);
    expect(kids[0]).toMatchObject({ id: 1, name: 'آوا', nextVaccine: { status: 'overdue' } });
    expect(parseCompanionChildren(null)).toEqual([]);
  });
});

describe('toChildBody', () => {
  it('sends every key in snake case, trimming the name', () => {
    expect(
      toChildBody({
        name: ' آوا ',
        birthDate: '2026-06-12',
        sex: null,
        deliveryType: 'cesarean',
        birthWeightKg: 3.2,
        birthLengthCm: null,
        birthHeadCm: null,
      }),
    ).toEqual({
      name: 'آوا',
      birth_date: '2026-06-12',
      sex: null,
      delivery_type: 'cesarean',
      birth_weight_kg: 3.2,
      birth_length_cm: null,
      birth_head_cm: null,
    });
  });
});
