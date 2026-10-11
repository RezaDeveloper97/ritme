import { describe, expect, it } from 'vitest';

import { newTask, tasksOnDay, toDraft, toPayload } from './program';

describe('challenge program draft', () => {
  it('sends nothing for a legacy one-day challenge', () => {
    expect(toPayload(toDraft(null))).toEqual({});
  });

  it('builds the payload of an enabled program', () => {
    const d = toDraft(null);
    d.enabled = true;
    d.duration = '7';
    d.reminder = '21:30';
    d.modesInclude = ['cycle', 'ttc'];
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
