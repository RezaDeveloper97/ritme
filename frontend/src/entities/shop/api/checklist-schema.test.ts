import { describe, expect, it } from 'vitest';

import { shopChecklistSchema, supplyReminderSchema } from './checklist-schema';

describe('shopChecklistSchema', () => {
  it('parses groups and falls back on unknown item states', () => {
    const out = shopChecklistSchema.parse({
      due_date: '2026-12-11',
      weeks_left: 10,
      read_only: true,
      summary: { ready: 3, total: 8, not_needed: 1, percent: 38 },
      groups: [
        {
          key: 'sleep',
          title: 'Sleep',
          ready: 1,
          total: 2,
          items: [{ id: null, layette_item_id: 4, is_custom: false, title: 'Cot', state: 'weird' }],
        },
      ],
    });
    expect(out.read_only).toBe(true);
    expect(out.groups[0].items[0].state).toBe('todo');
    expect(out.groups[0].items[0].quantity).toBeNull();
  });
});

describe('supplyReminderSchema', () => {
  it('is off by default', () => {
    expect(supplyReminderSchema.parse({}).enabled).toBe(false);
  });
});
