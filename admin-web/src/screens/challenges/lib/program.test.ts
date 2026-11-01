import { describe, expect, it } from 'vitest';

import { newTask, normalizeStartRule, tasksOnDay, toDraft, toPayload } from './program';

describe('challenge program draft', () => {
  it('sends nothing for a legacy one-day challenge', () => {
    expect(toPayload(toDraft(null))).toEqual({});
  });

  it('builds the payload of an enabled program', () => {
    const d = toDraft(null);
    d.enabled = true;
    d.duration = '7';
    d.reminder = '21:30';
    d.minDays = '5';
    d.plusDays = '1';
    d.tasks = [{ ...newTask('1', '7'), title: { fa: 'کشش' } }, { ...newTask('4'), title: { fa: 'تنفس' } }];
    const p = toPayload(d);
    expect(p).toMatchObject({ duration_days: 7, reminder_time: '21:30', completion_min_days: 5, reward_plus_days: 1, starts_on: null });
    expect(p.tasks).toEqual([
      { day_from: 1, day_to: 7, title: { fa: 'کشش' }, sort_order: 0 },
      { day_from: 4, day_to: 4, title: { fa: 'تنفس' }, sort_order: 1 },
    ]);
  });

  it('keeps the id of a saved task and finds the tasks of a day', () => {
    const d = toDraft(null);
    d.tasks = [{ ...newTask('2', '3'), id: 9 }];
    d.enabled = true;
    expect((toPayload(d).tasks as { id?: number }[])[0]?.id).toBe(9);
    expect(tasksOnDay(d.tasks, 1)).toHaveLength(0);
    expect(tasksOnDay(d.tasks, 3)).toHaveLength(1);
  });

  it('only sends starts_on for a fixed-date start', () => {
    const d = toDraft(null);
    d.enabled = true;
    d.startsOn = '2026-11-01';
    expect(toPayload(d).starts_on).toBeNull();
    d.startRule = 'fixed_date';
    expect(toPayload(d).starts_on).toBe('2026-11-01');
  });
});

describe('challenge rules (T-M10-10 / T-M10-13)', () => {
  const facts = [
    { name: 'mode', type: 'enum' as const, enum: ['cycle', 'pregnancy'], unit: null, group: 'g', description: {}, member_safe: true, pending: null },
    { name: 'pregnancy_week', type: 'int' as const, enum: null, unit: null, group: 'g', description: {}, member_safe: true, pending: null },
  ];
  const row = {
    id: 1, slug: null, title: {}, description: {}, cycle_day_from: null, cycle_day_to: null, category: null, is_active: true, sort_order: 0,
    program: {
      duration_days: 7, start_rule: 'on_phase_start', starts_on: null, reminder_time: null, modes_include: [], modes_exclude: [],
      phases_include: [], phases_exclude: [], reward_badge_code: null, reward_badge_title: null, reward_plus_days: 0, completion_min_days: null,
      start_phase: 'luteal',
      eligibility: { all_of: [{ fact: 'mode', op: 'eq', value: 'cycle' }], any_of: [], exclude: [] },
      contraindications: { all_of: [], any_of: [[{ fact: 'pregnancy_week', op: 'between', value: [1, 40] }]], exclude: [] },
    },
    tasks: [],
  };

  it('round-trips eligibility, contraindications and the phase start', () => {
    const d = toDraft(row);
    expect(d.startRule).toBe('on_phase_start');
    expect(d.startPhase).toBe('luteal');
    const p = toPayload(d, facts);
    expect(p.start_phase).toBe('luteal');
    expect(p.eligibility).toEqual({ all_of: [{ fact: 'mode', op: 'eq', value: 'cycle' }] });
    expect(p.contraindications).toEqual({ any_of: [[{ fact: 'pregnancy_week', op: 'between', value: [1, 40] }]] });
  });

  it('sends start_phase only for on_phase_start and maps the legacy anytime to on_join', () => {
    const d = toDraft(row);
    d.startRule = 'on_join';
    expect(toPayload(d, facts).start_phase).toBeNull();
    expect(normalizeStartRule('anytime')).toBe('on_join');
    expect(toDraft(null).startRule).toBe('on_join');
  });

  it('clears a rule with {} and keeps an unreadable stored contraindication when untouched', () => {
    const d = toDraft(null);
    d.enabled = true;
    expect(toPayload(d, facts).eligibility).toEqual({});
    expect(toPayload(d, facts).contraindications).toEqual({});
    d.contraindicationsInvalid = true;
    expect(toPayload(d, facts)).not.toHaveProperty('contraindications');
  });

  it('flags a rule the API could not read', () => {
    const bad = { ...row, program: { ...row.program, contraindications: { all_of: [], any_of: [], exclude: [], invalid: true } } };
    expect(toDraft(bad).contraindicationsInvalid).toBe(true);
  });
});
