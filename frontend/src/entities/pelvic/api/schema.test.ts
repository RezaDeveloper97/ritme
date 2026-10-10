import { describe, expect, it } from 'vitest';

import { pelvicAlertsSchema, pelvicDiarySchema, pelvicOverviewSchema } from './schema';

// Shapes of backend-go/contract/golden/pelvic (program_flow, diary_round_trip, show_empty).
const LEVEL = {
  code: 'level_2', number: 2, title: 'Level 2', body: 'Squeeze…', hold_sec: 5, rest_sec: 5, reps: 10, sets: 3,
  session_sec: 300, needs_review: true,
};
const DIARY = { date: '2026-09-23', leak: null, night_voids: null, uti_symptoms: [], uti_alert: false };

describe('pelvic schemas', () => {
  it('parses the overview with a program', () => {
    const o = pelvicOverviewSchema.parse({
      program: {
        started_on: '2026-09-09', week: 3, weeks_total: 8, completed: false, level: LEVEL, streak_days: 1,
        today_done: true, week_days: [{ date: '2026-09-23', done: true }, { date: 'bad' }],
      },
      diary_today: DIARY,
    });
    expect(o.program).toMatchObject({ week: 3, streakDays: 1, todayDone: true, weekDays: [{ date: '2026-09-23', done: true }] });
    expect(o.program?.level).toMatchObject({ number: 2, holdSec: 5, restSec: 5, reps: 10, sets: 3, sessionSec: 300 });
    expect(o.diaryToday).toEqual({ date: '2026-09-23', leak: null, nightVoids: null, utiSymptoms: [], utiAlert: false });
  });

  it('keeps no program and drops a broken level', () => {
    expect(pelvicOverviewSchema.parse({ program: null, diary_today: DIARY }).program).toBeNull();
    const o = pelvicOverviewSchema.parse({
      program: { started_on: '2026-09-09', week: 1, level: { code: 'x' }, week_days: [] },
      diary_today: DIARY,
    });
    expect(o.program?.level).toBeNull();
  });

  it('filters unknown diary values', () => {
    const d = pelvicDiarySchema.parse({ ...DIARY, leak: 'cough', night_voids: 2, uti_symptoms: ['burning', 'x'], uti_alert: true });
    expect(d).toMatchObject({ leak: 'cough', nightVoids: 2, utiSymptoms: ['burning'], utiAlert: true });
    expect(pelvicDiarySchema.parse({ ...DIARY, leak: 'odd' }).leak).toBeNull();
  });

  it('reads the alert copy', () => {
    expect(
      pelvicAlertsSchema.parse({ items: [{ code: 'uti_warning', title: 'See a doctor' }, { code: 'program_suitability', title: '' }] }),
    ).toEqual({ utiWarning: 'See a doctor', suitability: null });
    expect(pelvicAlertsSchema.parse(null)).toEqual({ utiWarning: null, suitability: null });
  });
});
