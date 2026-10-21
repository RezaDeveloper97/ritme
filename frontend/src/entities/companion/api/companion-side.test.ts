import { describe, expect, it } from 'vitest';

import { companionHomeSchema } from './companion-side';

/*
 * T-M9-02: the companion-side parser with the new (Go) member payload — no
 * mood, no free text, no `days_late` — and with the old (Laravel) one that
 * still carries them. Neither crashes, and nothing late-related or free-text
 * reaches the parsed model.
 */

const link = {
  id: 7,
  type: 'partner',
  owner: { id: 3, name: 'Sara' },
  accepted_at: '2026-09-01T09:00:00+03:30',
  grants: { cycle: 'view', symptoms: 'view', meds: 'view', appointments: 'view', pregnancy: 'none' },
  can_record_for: [],
  family: null,
};

function home(partner: Record<string, unknown>) {
  return {
    mode: 'companion',
    viewer: { id: 9, name: 'Ali' },
    has_partners: true,
    partners: [{ link, partner_name: 'Sara', phase: 'luteal', note: null, tips: [], child: null, ...partner }],
    empty_state: null,
    articles: [],
  };
}

const newPayload = home({
  cycle: {
    date: '2026-09-23',
    has_data: true,
    cycle_day: null,
    cycle_length: 28,
    main_phase: 'period_expected',
    days_to_period: null,
    predicted_next_period_start: null,
    confidence: 'medium',
  },
  pregnancy: null,
  symptoms: { from: '2026-09-17', to: '2026-09-23', days: [{ date: '2026-09-22', categories: { pain: {} } }] },
  meds: [{ id: 1, type: 'medication', title: 'Iron', form: 'tablet', dose: null, unit: null, times: ['21:00'], is_active: true }],
  appointments: [
    {
      id: 2,
      type: 'appointment',
      title: 'Checkup',
      kind: 'in_person',
      with: 'Dr. Rahimi',
      location: 'Clinic A',
      scheduled_at: '2026-09-30 10:00:00',
      days_until: 7,
      status: 'scheduled',
    },
  ],
});

const oldPayload = home({
  cycle: {
    date: '2026-09-23',
    has_data: true,
    cycle_day: 37,
    cycle_length: 28,
    main_phase: 'period_expected',
    days_to_period: 0,
    days_late: 9,
    predicted_next_period_start: '2026-09-14',
  },
  pregnancy: { is_active: false },
  symptoms: { days: [{ date: '2026-09-22', categories: { mood: { moods: ['calm'] } } }] },
  meds: [{ id: 1, title: 'Iron', notes: 'private note', subtitle: 'x', times: ['21:00'] }],
  appointments: [
    { id: 2, title: 'Checkup', notes: 'n', topic: 'ultrasound', result_note: 'r', prep: [{ id: 'p1', text: 't' }], scheduled_at: '2026-09-30 10:00:00', days_until: 7 },
  ],
});

describe('companionHomeSchema (member side, T-M9-02)', () => {
  it('parses the new payload without days_late, mood or free text', () => {
    const parsed = companionHomeSchema.parse(newPayload);
    const p = parsed.partners[0]!;
    expect(p.cycle).toEqual({
      date: '2026-09-23',
      hasData: true,
      cycleDay: null,
      cycleLength: 28,
      mainPhase: 'period_expected',
      daysToPeriod: null,
      predictedNextPeriodStart: null,
    });
    expect(p.symptomDays).toBe(1);
    expect(p.meds).toEqual([{ id: 1, title: 'Iron', times: ['21:00'] }]);
    expect(p.appointments).toEqual([{ id: 2, title: 'Checkup', scheduledAt: '2026-09-30 10:00:00', daysUntil: 7 }]);
  });

  it('keeps the old payload working and drops lateness and free text from it', () => {
    const parsed = companionHomeSchema.parse(oldPayload);
    const p = parsed.partners[0]!;
    expect(p.cycle).not.toHaveProperty('daysLate');
    expect(p.cycle?.cycleDay).toBe(37);
    expect(p.pregnancy).toEqual({ isActive: false, currentWeek: null, trimester: null });
    const text = JSON.stringify(parsed);
    for (const hidden of ['days_late', 'daysLate', 'private note', 'ultrasound', 'result_note', '"notes"', '"prep"']) {
      expect(text).not.toContain(hidden);
    }
    expect(p.meds).toEqual([{ id: 1, title: 'Iron', times: ['21:00'] }]);
  });

  it('reads a missing or malformed section as not shared, never a crash', () => {
    const parsed = companionHomeSchema.parse(home({ cycle: { has_data: 'x' }, meds: 'nope', symptoms: undefined }));
    const p = parsed.partners[0]!;
    expect(p.cycle).toBeNull();
    expect(p.meds).toBeNull();
    expect(p.symptomDays).toBeNull();
  });
});
