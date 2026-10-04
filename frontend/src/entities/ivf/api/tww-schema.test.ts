import { describe, expect, it } from 'vitest';

import { ivfDangerSignsSchema, ivfOutcomeSchema, ivfTwwSchema } from './tww-schema';

const step = (stage: string, status: string, date: string | null = null) => ({ stage, status, date });

// Shape of backend-go/contract/golden/ivf/tww_flow (CB-IVF-01).
const TWW = {
  cycle: {
    id: 1,
    number: 1,
    protocol: null,
    stage: 'tww',
    status: 'open',
    started_on: '2026-09-01',
    stim_started_on: '2026-09-03',
    retrieval_at: '2026-09-16 08:00:00',
    transfer_at: '2026-09-21 11:00:00',
    beta_on: '2026-10-02',
    next_scan_at: null,
    notify_companion: false,
    outcome: null,
    outcome_on: null,
    stage_day: 3,
    days_to_beta: 9,
    timeline: [
      step('prep', 'done', '2026-09-01'),
      step('stim', 'done', '2026-09-03'),
      step('retrieval', 'done', '2026-09-16'),
      step('transfer', 'done', '2026-09-21'),
      step('tww', 'current', '2026-09-21'),
      step('test', 'todo', '2026-10-02'),
    ],
  },
  transfer_on: '2026-09-21',
  beta_on: '2026-10-02',
  days_since_transfer: 2,
  days_to_beta: 9,
  today: { date: '2026-09-23', mood: 'hopeful' },
  moods: [{ date: '2026-09-23', mood: 'hopeful' }],
  luteal_support: [
    {
      med_id: 1,
      name: 'Progesterone',
      dose: '400',
      unit: 'mg',
      route: 'vaginal',
      doses: [
        { slot: '08:00', taken: true },
        { slot: '20:00', taken: false },
      ],
      taken: 1,
      total: 2,
    },
  ],
};

describe('ivfTwwSchema', () => {
  it('maps the golden two-week wait', () => {
    const tww = ivfTwwSchema.parse(TWW);
    expect(tww.cycle?.stage).toBe('tww');
    expect(tww.transferOn).toBe('2026-09-21');
    expect(tww.daysSinceTransfer).toBe(2);
    expect(tww.daysToBeta).toBe(9);
    expect(tww.today).toBe('2026-09-23');
    expect(tww.todayMood).toBe('hopeful');
    expect(tww.lutealSupport).toEqual([
      {
        medId: 1,
        name: 'Progesterone',
        dose: '400',
        unit: 'mg',
        route: 'vaginal',
        slots: [
          { slot: '08:00', taken: true },
          { slot: '20:00', taken: false },
        ],
        taken: 1,
        total: 2,
      },
    ]);
  });

  it('maps the empty state (no open cycle)', () => {
    const tww = ivfTwwSchema.parse({
      cycle: null,
      transfer_on: null,
      beta_on: null,
      days_since_transfer: null,
      days_to_beta: null,
      today: { date: '2026-09-23', mood: null },
      moods: [],
      luteal_support: [],
    });
    expect(tww.cycle).toBeNull();
    expect(tww.todayMood).toBeNull();
    expect(tww.lutealSupport).toEqual([]);
  });

  it('drops an unknown mood and a malformed medicine instead of failing', () => {
    const tww = ivfTwwSchema.parse({
      ...TWW,
      today: { date: '2026-09-23', mood: 'ecstatic' },
      moods: [{ date: '2026-09-22', mood: 'ecstatic' }, ...TWW.moods],
      luteal_support: [{ name: 'broken' }, ...TWW.luteal_support],
    });
    expect(tww.todayMood).toBeNull();
    expect(tww.moods).toHaveLength(1);
    expect(tww.lutealSupport).toHaveLength(1);
  });
});

describe('ivfOutcomeSchema', () => {
  it('maps a negative result with its next steps', () => {
    const out = ivfOutcomeSchema.parse({
      cycle: { ...TWW.cycle, status: 'closed', outcome: 'negative', outcome_on: '2026-09-23' },
      next_steps: ['loss', 'new_cycle'],
    });
    expect(out).toEqual({ result: 'negative', outcomeOn: '2026-09-23', nextSteps: ['loss', 'new_cycle'] });
  });

  it('keeps only known next steps', () => {
    const out = ivfOutcomeSchema.parse({
      cycle: { outcome: 'positive', outcome_on: '2026-09-22' },
      next_steps: ['pregnancy_setup', 'party'],
    });
    expect(out.nextSteps).toEqual(['pregnancy_setup']);
  });
});

describe('ivfDangerSignsSchema', () => {
  it('maps catalog rows with their hotlines', () => {
    const signs = ivfDangerSignsSchema.parse({
      items: [
        { code: 'ohss', title: 'Tell your doctor', body: 'Severe bloating', meta: { hotlines: ['115'] } },
        { code: 'fever_after_procedure', title: 'Fever', body: null, meta: null },
        { title: 'no code' },
      ],
    });
    expect(signs).toEqual([
      { code: 'ohss', title: 'Tell your doctor', body: 'Severe bloating', hotlines: ['115'] },
      { code: 'fever_after_procedure', title: 'Fever', body: null, hotlines: [] },
    ]);
  });

  it('falls back to an empty list', () => {
    expect(ivfDangerSignsSchema.parse(null)).toEqual([]);
  });
});
