import { describe, expect, it } from 'vitest';

import { contraceptionOverviewSchema } from './schema';

/* Fixtures follow backend-go/contract/golden/contraception/{show_empty,pill_flow,iud_flow}.en.json. */

const days = Array.from({ length: 28 }, (_, i) => ({
  day: i + 1,
  date: `2026-10-${String(i + 1).padStart(2, '0')}`,
  kind: i < 21 ? 'active' : 'break',
  status: i < 21 ? (i < 7 ? 'taken' : i === 7 ? 'pending' : 'upcoming') : null,
}));

const pillOverview = {
  tracking: true,
  method: {
    method: 'combined_pill',
    pack_type: '21_7',
    pack_started_on: '2026-10-01',
    packs_left: 1,
    reminder: { enabled: true, time: '21:00' },
    inserted_on: null,
    iud_lifetime_years: null,
    followup_on: null,
    followup_done: false,
    iud_replace_on: null,
    injected_on: null,
    next_injection_on: null,
    replace_on: null,
  },
  pill: {
    pack_number: 1,
    pack_day: 8,
    pack_week: 2,
    pack_length: 28,
    active_days: 21,
    today: { date: '2026-10-08', kind: 'active', status: 'pending' },
    days,
    streak_days: 7,
    missed_count: 0,
    next_pack_on: '2026-10-29',
    packs_left: 1,
    runs_out_on: '2026-11-26',
    refill_on: '2026-11-21',
  },
  reminders: [
    {
      kind: 'pill_refill',
      reminder_id: 12,
      type: 'custom',
      title: 'Buy your next pill pack',
      due_on: '2026-11-21',
      scheduled_at: '2026-11-21 09:00:00',
      recurrence: 'none',
      is_active: true,
    },
  ],
};

describe('contraceptionOverviewSchema', () => {
  it('parses the empty screen', () => {
    expect(contraceptionOverviewSchema.parse({ tracking: false, method: null, pill: null, reminders: [] })).toEqual({
      tracking: false,
      method: null,
      pill: null,
      reminders: [],
    });
  });

  it('maps a pill method and its pack to camelCase', () => {
    const o = contraceptionOverviewSchema.parse(pillOverview);
    expect(o.method).toMatchObject({ method: 'combined_pill', packType: '21_7', packStartedOn: '2026-10-01', packsLeft: 1 });
    expect(o.method?.reminder).toEqual({ enabled: true, time: '21:00' });
    expect(o.pill).toMatchObject({ packDay: 8, activeDays: 21, streakDays: 7, nextPackOn: '2026-10-29', refillOn: '2026-11-21' });
    expect(o.pill?.days).toHaveLength(28);
    expect(o.pill?.days[21]).toEqual({ day: 22, date: '2026-10-22', kind: 'break', status: null });
    expect(o.reminders[0]).toMatchObject({ kind: 'pill_refill', reminderId: 12, isActive: true });
  });

  it('parses an IUD without a pack', () => {
    const o = contraceptionOverviewSchema.parse({
      ...pillOverview,
      method: {
        ...pillOverview.method,
        method: 'copper_iud',
        pack_type: null,
        pack_started_on: null,
        packs_left: null,
        reminder: null,
        inserted_on: '2026-09-01',
        iud_lifetime_years: 10,
        followup_on: '2026-10-13',
        iud_replace_on: '2036-09-01',
      },
      pill: null,
      reminders: [],
    });
    expect(o.method).toMatchObject({ method: 'copper_iud', insertedOn: '2026-09-01', iudLifetimeYears: 10, reminder: null });
    expect(o.pill).toBeNull();
  });

  it('rejects an unknown method', () => {
    expect(() =>
      contraceptionOverviewSchema.parse({ ...pillOverview, method: { ...pillOverview.method, method: 'magic' } }),
    ).toThrow();
  });
});
