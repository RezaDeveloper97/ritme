import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

import { defaultGivenOn, givenDoses, isActionable, notedDoses } from '../model/schedule';
import { vaccineScheduleSchema } from './schema';

/* Boundary contract of /children/{id}/vaccines* against the Go goldens (B-N5-02). */
const GOLDEN_DIR = resolve(process.cwd(), '../backend-go/contract/golden/children');
interface Golden {
  steps: { request: { method: string; url: string }; status: number; body: { data?: unknown } }[];
}
const step = (method: string, url: string): unknown =>
  (JSON.parse(readFileSync(resolve(GOLDEN_DIR, 'vaccines.fa.json'), 'utf8')) as Golden).steps.find(
    (s) => s.request.method === method && s.request.url === url && s.status < 300,
  )?.body.data;

describe.skipIf(!existsSync(GOLDEN_DIR))('vaccine schedule parser', () => {
  it('parses the fresh schedule', () => {
    const s = vaccineScheduleSchema.parse(step('GET', '/api/v1/children/1/vaccines'));
    expect(s.summary).toMatchObject({ given: 0, total: 21, completedVisits: 0, complete: false });
    expect(s.summary.next).toMatchObject({ code: 'birth', status: 'overdue' });
    expect(s.visits.length).toBeGreaterThan(5);
    expect(s.visits[0]?.doses[0]).toMatchObject({ code: 'bcg', givenOn: null, note: null });
    expect(s.reminderLabel).toBeTruthy();
    expect(s.note).toBeTruthy();
    expect(givenDoses(s.visits)).toEqual([]);
  });

  it('parses a marked visit with its note', () => {
    const s = vaccineScheduleSchema.parse(step('POST', '/api/v1/children/1/vaccines/visits/m2'));
    expect(s.summary).toMatchObject({ given: 7, completedVisits: 2, upToDate: true });
    expect(s.summary.next).toMatchObject({ code: 'm4', status: 'soon', daysLeft: 3 });
    const m2 = s.visits.find((v) => v.code === 'm2');
    expect(m2?.status).toBe('done');
    expect(m2?.doses.every((d) => d.givenOn === '2026-07-27')).toBe(true);
    expect(givenDoses(s.visits)).toHaveLength(7);
    const noted = notedDoses(s.visits);
    expect(noted).toHaveLength(4);
    expect(noted[0]?.dose.note).toBe('تب خفیف');
    expect(isActionable(s.visits.find((v) => v.code === 'm4')!, 'm4')).toBe(true);
    expect(isActionable(m2!, 'm4')).toBe(false);
  });

  it('parses a single dose mark and unmark', () => {
    const put = vaccineScheduleSchema.parse(step('PUT', '/api/v1/children/1/vaccines/penta_2'));
    expect(put.summary.given).toBe(8);
    const del = vaccineScheduleSchema.parse(step('DELETE', '/api/v1/children/1/vaccines/penta_2'));
    expect(del.summary.given).toBe(7);
  });
});

describe('vaccine schedule helpers', () => {
  it('defaults the shot date to the passed due date, else today, never before birth', () => {
    expect(defaultGivenOn({ dueDate: '2026-07-26' }, '2026-09-23', '2026-05-26')).toBe('2026-07-26');
    expect(defaultGivenOn({ dueDate: '2026-09-26' }, '2026-09-23', '2026-05-26')).toBe('2026-09-23');
    expect(defaultGivenOn({ dueDate: '2026-05-01' }, '2026-09-23', '2026-05-26')).toBe('2026-05-26');
  });
});
