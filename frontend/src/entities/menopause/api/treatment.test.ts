import { describe, expect, it } from 'vitest';

import { toTreatmentItemBody, treatmentItemInput } from '../model/treatment';
import { treatmentScreenSchema } from './treatment';

const week = (taken: string[] = []) =>
  ['2026-09-19', '2026-09-20', '2026-09-21', '2026-09-22', '2026-09-23', '2026-09-24', '2026-09-25'].map((date) => ({
    date,
    taken: taken.includes(date),
    amount: null,
  }));

// Trimmed from backend-go/contract/golden/menopause/treatment_flow.json.
const SCREEN = {
  date: '2026-09-23',
  week: { from: '2026-09-19', to: '2026-09-25' },
  items: {
    hrt: [
      {
        id: 1,
        kind: 'hrt',
        name: 'Estrogen gel',
        dose: '1 pump',
        schedule: 'morning',
        form: 'tablet',
        started_on: '2026-08-01',
        review_on: '2026-11-21',
        stopped_on: null,
        active: true,
        taken_today: true,
        week: week(['2026-09-23']),
        days_taken: 1,
        days: 5,
        adherence_pct: 20,
        goal: null,
        reminder: { id: 101805, time: '08:00', notify: true, active: true },
      },
      { id: 9, kind: 'unknown', name: 'dropped' },
    ],
    supplement: [],
    lifestyle: [
      {
        id: 3,
        kind: 'lifestyle',
        name: 'Brisk walk',
        dose: null,
        schedule: null,
        form: null,
        started_on: '2026-09-01',
        review_on: null,
        stopped_on: null,
        active: true,
        taken_today: true,
        week: week(),
        days_taken: 1,
        days: 5,
        adherence_pct: null,
        goal: { target: 150, unit: 'minutes', amount: 30, done: false },
        reminder: null,
      },
    ],
  },
  stopped: [],
  review: { on: '2026-11-21', item_id: 1, suggested: false },
  side_effects: {
    codes: ['breast_tenderness', 'spotting', 'headache', 'bloating', 'mood_change'],
    today: ['breast_tenderness', 'spotting', 'not_a_code'],
    week: [{ date: '2026-09-23', codes: ['spotting'] }],
  },
  tips: [
    {
      code: 'lifestyle_brisk_walk',
      title: 'Brisk walking',
      body: '150 minutes a week',
      meta: { placement: 'treatment_lifestyle', weekly_goal: 150, goal_unit: 'minutes' },
    },
    { code: 'doctor_only', title: 'Only with your doctor', body: 'Start, stop…', meta: { placement: 'treatment' } },
  ],
};

describe('treatment screen schema (CB-MENO-10)', () => {
  it('parses items by kind, review, side effects and tips', () => {
    const s = treatmentScreenSchema.parse(SCREEN);
    expect(s.items.hrt).toHaveLength(1);
    expect(s.items.hrt[0]).toMatchObject({ id: 1, startedOn: '2026-08-01', takenToday: true, adherencePct: 20 });
    expect(s.items.hrt[0]?.reminder?.notify).toBe(true);
    expect(s.items.lifestyle[0]?.goal).toEqual({ target: 150, unit: 'minutes', amount: 30, done: false });
    expect(s.review).toEqual({ on: '2026-11-21', itemId: 1, suggested: false });
    expect(s.sideEffects.today).toEqual(['breast_tenderness', 'spotting']);
    expect(s.tips[0]).toMatchObject({ placement: 'treatment_lifestyle', weeklyGoal: 150, goalUnit: 'minutes' });
    expect(s.tips[1]?.weeklyGoal).toBeNull();
  });

  it('survives an empty screen', () => {
    const s = treatmentScreenSchema.parse({ date: '2026-09-23', week: { from: '2026-09-19', to: '2026-09-25' } });
    expect(s.items).toEqual({ hrt: [], supplement: [], lifestyle: [] });
    expect(s.review).toBeNull();
    expect(s.sideEffects.codes).toHaveLength(5);
  });

  it('builds the write body per kind (PUT = full replace)', () => {
    const s = treatmentScreenSchema.parse(SCREEN);
    const hrt = treatmentItemInput(s.items.hrt[0]!);
    expect(toTreatmentItemBody({ ...hrt, stoppedOn: '2026-09-23' }, false)).toEqual({
      name: 'Estrogen gel',
      dose: '1 pump',
      schedule: 'morning',
      form: 'tablet',
      remind: true,
      started_on: '2026-08-01',
      review_on: '2026-11-21',
      stopped_on: '2026-09-23',
    });
    const walk = treatmentItemInput(s.items.lifestyle[0]!);
    expect(toTreatmentItemBody(walk, true)).toEqual({
      kind: 'lifestyle',
      name: 'Brisk walk',
      weekly_goal: 150,
      goal_unit: 'minutes',
      started_on: '2026-09-01',
      review_on: null,
      stopped_on: null,
    });
  });
});
