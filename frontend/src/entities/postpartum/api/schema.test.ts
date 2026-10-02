import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

import { toRecoveryBody } from './queries';
import { epdsResultSchema, overviewSchema, questionnaireSchema, recoverySchema } from './schema';

/* Boundary contract of /api/v1/postpartum* against the Go contract goldens (B-N5-01). */
const GOLDEN_DIR = resolve(process.cwd(), '../backend-go/contract/golden/postpartum');
interface Golden {
  steps: { status: number; body: { data?: unknown } }[];
}
const step = (name: string, i: number): unknown =>
  (JSON.parse(readFileSync(resolve(GOLDEN_DIR, `${name}.json`), 'utf8')) as Golden).steps[i].body.data;

describe.skipIf(!existsSync(GOLDEN_DIR))('postpartum parsers', () => {
  it('parses an active overview (17 days, week 3, full check due)', () => {
    const o = overviewSchema.parse(step('activate_direct', 0));
    expect(o.active).toBe(true);
    expect(o.setupRequired).toBe(false);
    expect(o.profile).toMatchObject({ birthDate: '2026-09-06', deliveryType: 'vaginal', babyCount: 1 });
    expect(o.status).toMatchObject({ daysSinceBirth: 17, weeks: 2, days: 3, week: 3, puerperiumDays: 42 });
    expect(o.checkin?.due).toBe('full');
    expect(o.today?.breasts).toBeNull();
    expect(o.today?.painLocations).toEqual([]);
    expect(o.alerts[0]).toMatchObject({ key: 'checkin_full', action: { type: 'open_check', kind: 'full' } });
    expect(o.weekTip?.week).toBe(3);
    expect(o.callWhen?.title).toBeTruthy();
  });

  it('parses an inactive overview — call-when only', () => {
    const o = overviewSchema.parse(step('show_inactive.none', 0));
    expect(o).toMatchObject({ active: false, mode: 'cycle', profile: null, status: null, alerts: [] });
    expect(o.callWhen?.body).toBeTruthy();
  });

  it('parses recovery with the heavy-bleeding call alert', () => {
    const r = recoverySchema.parse(step('recovery_flow.en', 1));
    expect(r).toMatchObject({
      lochiaAmount: 'heavy',
      lochiaColor: 'red',
      painLevel: 'moderate',
      painLocations: ['stitches', 'abdomen'],
      breasts: ['engorgement'],
      feedsCount: 8,
      sleepHours: 4.5,
    });
    expect(r.alerts[0].action).toEqual({ type: 'call', number: '115', label: null });
    const cleared = recoverySchema.parse(step('recovery_flow.en', 2));
    expect(cleared.breasts).toEqual([]);
    expect(cleared.sleepHours).toBeNull();
  });

  it('parses the short questionnaire (items 3/4/5 with option scores)', () => {
    const q = questionnaireSchema.parse(step('questions_short.none', 0));
    expect(q.kind).toBe('short');
    expect(q.max).toBe(9);
    expect(q.items.map((i) => i.code)).toEqual(['q3', 'q4', 'q5']);
    expect(q.items[0].options.map((o) => o.score)).toEqual([3, 2, 1, 0]);
    expect(q.disclaimer).toBeTruthy();
  });

  it('parses the full questionnaire (10 items)', () => {
    const q = questionnaireSchema.parse(step('questions_full', 0));
    expect(q.items).toHaveLength(10);
    expect(q.items[9].code).toBe('q10');
  });

  it('keeps the urgent safety message with its three call actions', () => {
    const r = epdsResultSchema.parse(step('epds_urgent_self_harm.none', 1));
    expect(r.check).toMatchObject({ kind: 'full', total: 2, urgent: true });
    expect(r.safety?.level).toBe('urgent');
    expect(r.safety?.actions.map((a) => (a.type === 'call' ? a.number : a.kind))).toEqual(['115', '123', '1480']);
  });

  it('parses the short follow-up result', () => {
    const r = epdsResultSchema.parse(step('epds_short_follow_up.none', 1));
    expect(r.followUp).toBe('full');
    expect(r.safety).toMatchObject({ level: 'follow_up', actions: [{ type: 'open_check', kind: 'full' }] });
    expect(r.checkin?.due).toBe('full');
  });
});

describe('safety parsing never drops an urgent message', () => {
  const check = { id: 1, kind: 'full', taken_on: '2026-09-23', week: 4, total: 14, max: 30, band: 'likely', urgent: true };
  it('a malformed safety block still comes back as urgent', () => {
    const r = epdsResultSchema.parse({ check, safety: { level: 'urgent', actions: 'oops' }, follow_up: null });
    expect(r.safety).toMatchObject({ level: 'urgent', actions: [] });
  });
  it('bad actions are dropped, good ones kept', () => {
    const r = epdsResultSchema.parse({
      check,
      safety: { level: 'urgent', title: 't', body: 'b', actions: [{ type: 'call', number: 'x' }, { type: 'call', number: '115' }] },
    });
    expect(r.safety?.actions).toEqual([{ type: 'call', number: '115', label: null }]);
  });
});

describe('toRecoveryBody', () => {
  it('sends only the keys present, null clears', () => {
    expect(toRecoveryBody({ lochiaAmount: 'light', sleepHours: null })).toEqual({ lochia_amount: 'light', sleep_hours: null });
    expect(toRecoveryBody({})).toEqual({});
  });
});
