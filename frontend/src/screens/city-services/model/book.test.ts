import { describe, expect, it } from 'vitest';

import type { CityServiceSlot } from '@/entities/city-service';

import { ageFits, bookingWhen, groupSlotsByDay, totalMonths } from './book';

const slot = (id: number, starts_at: string): CityServiceSlot => ({ id, service_id: 1, starts_at, ends_at: null, capacity: 4, seats_left: 2, is_full: false });

describe('groupSlotsByDay', () => {
  it('groups by day, soonest first, and drops unparsable stamps', () => {
    const days = groupSlotsByDay([slot(3, '2026-10-13T09:00:00+03:30'), slot(1, '2026-10-12 17:00:00'), slot(2, '2026-10-12T10:30:00'), slot(9, 'nope')]);
    expect(days.map((d) => d.day)).toEqual(['2026-10-12', '2026-10-13']);
    expect(days[0].slots.map((s) => s.time)).toEqual(['10:30', '17:00']);
  });
});

describe('ageFits / totalMonths', () => {
  it('uses the API total of completed months (years are not added again)', () => expect(totalMonths({ years: 1, months: 14 })).toBe(14));
  it('checks the declared range, no range fits', () => {
    expect(ageFits(14, { age_min_months: 6, age_max_months: 18 })).toBe(true);
    expect(ageFits(30, { age_min_months: 6, age_max_months: 18 })).toBe(false);
    expect(ageFits(3, { age_min_months: null, age_max_months: null })).toBe(true);
    expect(ageFits(3, undefined)).toBe(true);
  });
});

describe('bookingWhen', () => {
  it('prefers the slot, falls back to the preferred day', () => {
    expect(bookingWhen({ preferred_date: '2026-10-12', slot: { starts_at: '2026-10-14T17:00:00' } })).toEqual({ day: '2026-10-14', time: '17:00' });
    expect(bookingWhen({ preferred_date: '2026-10-12T00:00:00Z', slot: null })).toEqual({ day: '2026-10-12', time: null });
  });
});
