import { describe, expect, it } from 'vitest';

import { ivfHomeSchema, ivfMedsTodaySchema, ivfStagesSchema } from './schema';

const step = (stage: string, status: string, date: string | null = null) => ({ stage, status, date });

// Shape of backend-go/contract/golden/ivf/cycle_flow + meds_flow (CB-IVF-01).
const HOME = {
  enabled: true,
  cycle: {
    id: 1,
    number: 1,
    protocol: 'antagonist',
    stage: 'stim',
    status: 'open',
    started_on: '2026-09-10',
    stim_started_on: '2026-09-17',
    retrieval_at: null,
    transfer_at: null,
    beta_on: '2026-10-10',
    next_scan_at: '2026-09-24 09:00:00',
    notify_companion: false,
    outcome: null,
    outcome_on: null,
    stage_day: 7,
    days_to_beta: 17,
    timeline: [
      step('prep', 'done', '2026-09-10'),
      step('stim', 'current', '2026-09-17'),
      step('retrieval', 'todo'),
      step('transfer', 'todo'),
      step('tww', 'todo'),
      step('test', 'todo', '2026-10-10'),
    ],
  },
  cycles_count: 1,
  today: {
    date: '2026-09-23',
    doses: [
      {
        med_id: 2,
        reminder_id: 101806,
        name: 'Antagonist',
        dose: null,
        unit: null,
        route: 'subcutaneous',
        role: 'suppression',
        is_trigger: false,
        date: '2026-09-23',
        slot: '08:00',
        taken: true,
        site: 'abdomen_upper_right',
      },
      {
        med_id: 1,
        reminder_id: 101805,
        name: 'FSH',
        dose: '150',
        unit: 'iu',
        route: 'subcutaneous',
        role: 'stimulation',
        is_trigger: false,
        date: '2026-09-23',
        slot: '20:00',
        taken: false,
        site: null,
      },
    ],
  },
  next_appointment: {
    kind: 'scan',
    appointment: {
      id: 101805,
      type: 'appointment',
      title: 'سونوگرافی فولیکول',
      scheduled_at: '2026-09-24 09:00:00',
      days_until: 1,
      prep: [{ id: 'p1', text: 'ناشتا نیاز نیست', done: false }],
    },
  },
  latest_scan: null,
  companion: { linked: true, notify: false, shares_meds: false, shares_appointments: false },
};

describe('ivfHomeSchema', () => {
  it('maps the open cycle, today and the next appointment', () => {
    const home = ivfHomeSchema.parse(HOME);
    expect(home.enabled).toBe(true);
    expect(home.cycle?.stage).toBe('stim');
    expect(home.cycle?.stageDay).toBe(7);
    expect(home.cycle?.timeline.map((s) => s.status)).toEqual(['done', 'current', 'todo', 'todo', 'todo', 'todo']);
    expect(home.today.doses).toHaveLength(2);
    expect(home.today.doses[1]).toMatchObject({ medId: 1, dose: '150', unit: 'iu', taken: false, slot: '20:00' });
    expect(home.nextAppointment).toEqual({
      kind: 'scan',
      id: 101805,
      title: 'سونوگرافی فولیکول',
      scheduledAt: '2026-09-24 09:00:00',
      daysUntil: 1,
      prep: 'ناشتا نیاز نیست',
    });
    expect(home.companion).toEqual({ linked: true, notify: false });
  });

  it('reads the empty home (no cycle, switch off)', () => {
    const home = ivfHomeSchema.parse({
      enabled: false,
      cycle: null,
      cycles_count: 0,
      today: { date: '2026-09-23', doses: [] },
      next_appointment: null,
      latest_scan: null,
      companion: { linked: false, notify: false, shares_meds: false, shares_appointments: false },
    });
    expect(home).toMatchObject({ enabled: false, cycle: null, nextAppointment: null });
  });

  it('drops a malformed dose or step instead of the whole home', () => {
    const broken = structuredClone(HOME) as typeof HOME & { today: { doses: unknown[] } };
    broken.today.doses.push({ med_id: 'x' });
    (broken.cycle.timeline as unknown[]).push({ stage: 'nope' });
    const home = ivfHomeSchema.parse(broken);
    expect(home.today.doses).toHaveLength(2);
    expect(home.cycle?.timeline).toHaveLength(6);
  });

  it('accepts a numeric dose and unknown route as other', () => {
    const odd = structuredClone(HOME);
    (odd.today.doses[1] as Record<string, unknown>).dose = 225;
    (odd.today.doses[1] as Record<string, unknown>).route = 'nasal';
    const dose = ivfHomeSchema.parse(odd).today.doses[1];
    expect(dose?.dose).toBe('225');
    expect(dose?.route).toBe('other');
  });
});

describe('ivfMedsTodaySchema', () => {
  it('takes `today` out of the meds view', () => {
    expect(ivfMedsTodaySchema.parse({ cycle: null, today: HOME.today, meds: [] }).doses).toHaveLength(2);
  });
});

describe('ivfStagesSchema', () => {
  it('keeps code/title/body and drops malformed items', () => {
    expect(
      ivfStagesSchema.parse({ items: [{ code: 'prep', title: 'آماده‌سازی', body: '' }, { title: 'x' }] }),
    ).toEqual([{ code: 'prep', title: 'آماده‌سازی', body: null }]);
    expect(ivfStagesSchema.parse(null)).toEqual([]);
  });
});
