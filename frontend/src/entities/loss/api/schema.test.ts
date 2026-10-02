import { describe, expect, it } from 'vitest';

import { lossCatalogSchema, lossNoteSchema, lossStateSchema } from './schema';

// Shapes from backend-go/contract/golden/loss/*.json.
const recorded = {
  loss: {
    id: 1,
    type: 'early_miscarriage',
    occurred_on: '2026-09-20',
    notify_companion: false,
    companion_notified: false,
    content_stopped: true,
    next_step: null,
    created_at: '2026-09-23T10:00:00+03:30',
  },
  losses_count: 1,
  recurrent_hint: false,
  followup: {
    bleeding: { stopped: false, stopped_on: null, today: 'light' },
    beta: {
      negative: false,
      negative_on: null,
      next_on: '2026-09-30',
      appointment: { id: 101805, type: 'appointment', scheduled_at: '2026-09-30 09:00:00', status: 'scheduled' },
    },
    visit: { suggested_on: '2026-10-04', appointment: null },
  },
  mood: { today: 'numb', recent: [{ date: '2026-09-23', mood: 'numb' }, { date: '2026-09-22', mood: 'bogus' }] },
  note: { has_note: true, updated_at: '2026-09-23T10:00:00+03:30' },
};

describe('lossStateSchema', () => {
  it('reads an empty state (no loss recorded)', () => {
    const s = lossStateSchema.parse({
      loss: null,
      losses_count: 0,
      recurrent_hint: false,
      followup: null,
      mood: null,
      note: null,
    });
    expect(s).toEqual({ loss: null, lossesCount: 0, recurrentHint: false, followup: null, mood: null, note: null });
  });

  it('camel-cases a recorded loss and drops an unknown mood entry', () => {
    const s = lossStateSchema.parse(recorded);
    expect(s.loss).toMatchObject({ type: 'early_miscarriage', occurredOn: '2026-09-20', contentStopped: true, nextStep: null });
    expect(s.followup?.bleeding.today).toBe('light');
    expect(s.followup?.beta).toMatchObject({ nextOn: '2026-09-30', appointment: { id: 101805, scheduledAt: '2026-09-30 09:00:00' } });
    expect(s.followup?.visit).toEqual({ suggestedOn: '2026-10-04', appointment: null });
    expect(s.mood).toEqual({ today: 'numb', recent: [{ date: '2026-09-23', mood: 'numb' }] });
    expect(s.note).toEqual({ hasNote: true, updatedAt: '2026-09-23T10:00:00+03:30' });
  });

  it('degrades an unknown type to unspecified rather than failing', () => {
    const s = lossStateSchema.parse({ ...recorded, loss: { ...recorded.loss, type: 'new_kind', next_step: 'odd' } });
    expect(s.loss?.type).toBe('unspecified');
    expect(s.loss?.nextStep).toBeNull();
  });
});

describe('lossNoteSchema', () => {
  it('reads a note and a cleared one', () => {
    expect(lossNoteSchema.parse({ note: 'x', updated_at: '2026-09-23T10:00:00+03:30' })).toEqual({
      note: 'x',
      updatedAt: '2026-09-23T10:00:00+03:30',
    });
    expect(lossNoteSchema.parse({ note: null, updated_at: null })).toEqual({ note: null, updatedAt: null });
  });
});

describe('lossCatalogSchema', () => {
  it('keeps valid items and drops malformed ones', () => {
    const items = lossCatalogSchema.parse({
      group: 'loss_hotlines',
      items: [
        { code: 'emergency', title: 'اورژانس ۱۱۵', body: null, meta: { number: '115', kind: 'medical' } },
        { title: 'no code' },
      ],
    });
    expect(items).toEqual([{ code: 'emergency', title: 'اورژانس ۱۱۵', body: null, meta: { number: '115', kind: 'medical' } }]);
  });

  it('survives a broken payload', () => {
    expect(lossCatalogSchema.parse(null)).toEqual([]);
  });
});
