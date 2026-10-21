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

describe('postpartum red flags and visits (T-M9-06)', () => {
  const day = {
    date: '2026-10-21',
    lochia_amount: null,
    lochia_color: null,
    pain_level: null,
    pain_locations: [],
    breasts: null,
    feeds_count: null,
    sleep_hours: null,
    temperature_c: 38,
    pad_soaked_within_hour: null,
    severe_pain: false,
    breathless: true,
    postpartum_visit: true,
    alerts: [
      { key: 'breathless', level: 'urgent', title: 'نفست تنگ است', body: 'همین حالا با ۱۱۵ تماس بگیر یا به اورژانس برو.', action_label: 'تماس با ۱۱۵', action: { type: 'call', number: '115' } },
    ],
  };

  it('parses the red flags and the urgent 115 alert', () => {
    const r = recoverySchema.parse(day);
    expect(r).toMatchObject({ temperatureC: 38, padSoakedWithinHour: null, severePain: false, breathless: true, postpartumVisit: true });
    expect(r.alerts[0]).toMatchObject({ key: 'breathless', level: 'urgent', action: { type: 'call', number: '115' } });
  });

  it('an older payload without the new keys reads as not answered', () => {
    const old: Record<string, unknown> = { ...day };
    delete old.temperature_c;
    delete old.breathless;
    const r = recoverySchema.parse(old);
    expect(r.temperatureC).toBeNull();
    expect(r.breathless).toBeNull();
  });

  it('parses the visits and drops a bad row', () => {
    const o = overviewSchema.parse({
      active: true,
      mode: 'postpartum',
      visits: [
        { kind: 'contact', due_on: '2026-10-22', done: true, done_on: '2026-10-15', status: 'done' },
        { kind: 'reminder', due_on: '2026-11-12', done: false, done_on: null, status: 'due' },
        { kind: 'nope', due_on: 'x', done: false, status: 'due' },
      ],
    });
    expect(o.visits).toEqual([
      { kind: 'contact', dueOn: '2026-10-22', done: true, doneOn: '2026-10-15', status: 'done' },
      { kind: 'reminder', dueOn: '2026-11-12', done: false, doneOn: null, status: 'due' },
    ]);
    expect(overviewSchema.parse({ active: false, mode: 'cycle' }).visits).toEqual([]);
  });

  it('sends the red flags in API keys', () => {
    expect(toRecoveryBody({ temperatureC: 38.2, padSoakedWithinHour: true, postpartumVisit: null })).toEqual({
      temperature_c: 38.2,
      pad_soaked_within_hour: true,
      postpartum_visit: null,
    });
  });
});
