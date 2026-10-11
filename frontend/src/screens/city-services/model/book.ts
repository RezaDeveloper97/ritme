import type { CityServiceService, CityServiceSlot } from '@/entities/city-service';

import { slotParts } from './place';

/* CB-DIR-09 — pure helpers of the booking request (Dir_Book): slots per day, her child's age vs the service. */

export interface SlotDay {
  day: string;
  slots: { slot: CityServiceSlot; time: string }[];
}

/** Open slots grouped per local day, soonest first; slots that cannot be parsed are dropped. */
export function groupSlotsByDay(slots: CityServiceSlot[]): SlotDay[] {
  const byDay = new Map<string, SlotDay>();
  // Sort on the parsed stamp: the API may use `T` or a space as the separator, which would not compare equal as text.
  const parsed = slots.flatMap((slot) => {
    const p = slotParts(slot.starts_at);
    return p ? [{ slot, p }] : [];
  });
  parsed.sort((a, b) => `${a.p.day} ${a.p.time}`.localeCompare(`${b.p.day} ${b.p.time}`));
  for (const { slot, p } of parsed) {
    const entry = byDay.get(p.day) ?? { day: p.day, slots: [] };
    entry.slots.push({ slot, time: p.time });
    byDay.set(p.day, entry);
  }
  return [...byDay.values()];
}

/** Total age in whole months of a child (`ChildAge` carries years + the remaining months). */
export const totalMonths = (age: { years: number; months: number }): number => age.years * 12 + age.months;

/** Whether a child's age fits the service's declared range (no range = fits). The server checks again. */
export function ageFits(months: number, s: Pick<CityServiceService, 'age_min_months' | 'age_max_months'> | undefined): boolean {
  if (!s) return true;
  if (s.age_min_months !== null && months < s.age_min_months) return false;
  if (s.age_max_months !== null && months > s.age_max_months) return false;
  return true;
}

/** The booking's day (`YYYY-MM-DD`) and, for a slot booking, its `HH:mm` — the slot wins over the preferred day. */
export function bookingWhen(b: { preferred_date: string; slot: { starts_at: string | null } | null }): { day: string; time: string | null } {
  const p = b.slot?.starts_at ? slotParts(b.slot.starts_at) : null;
  return p ? { day: p.day, time: p.time } : { day: b.preferred_date.slice(0, 10), time: null };
}
