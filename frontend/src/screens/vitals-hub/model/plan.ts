import { VITAL_TYPES, type PlanItem, type VitalType } from '@/entities/vital';

/*
 * Plan editor rules, mirroring PUT /vitals/plan (≤ 8 items, one per type +
 * slot, a slot of the item's type, ≥ 1 weekday). Pure.
 */

export const MAX_PLAN_ITEMS = 8;
export const ALL_DAYS: readonly number[] = [0, 1, 2, 3, 4, 5, 6];

export type PlanSlots = Record<VitalType, readonly string[]>;

const key = (i: Pick<PlanItem, 'type' | 'slot'>) => `${i.type}|${i.slot}`;

/** The first (type, slot) not in the plan yet, every day, no reminder; null when all are used. */
export function nextFreeItem(items: readonly PlanItem[], slots: PlanSlots): PlanItem | null {
  if (items.length >= MAX_PLAN_ITEMS) return null;
  const used = new Set(items.map(key));
  for (const type of VITAL_TYPES) {
    for (const slot of slots[type] ?? []) {
      if (!used.has(`${type}|${slot}`)) return { type, slot, days: ALL_DAYS, remindAt: null };
    }
  }
  return null;
}

/** An item after its type changes: keeps the slot when the new type has it, else that type's first free slot. */
export function withType(items: readonly PlanItem[], index: number, type: VitalType, slots: PlanSlots): PlanItem {
  const item = items[index];
  const used = new Set(items.filter((_, i) => i !== index).map(key));
  const list = slots[type] ?? [];
  const slot = list.includes(item.slot) && !used.has(`${type}|${item.slot}`) ? item.slot : (list.find((s) => !used.has(`${type}|${s}`)) ?? list[0] ?? item.slot);
  return { ...item, type, slot };
}

export function toggleDay(days: readonly number[], day: number): number[] {
  return days.includes(day) ? days.filter((d) => d !== day) : [...days, day].sort((a, b) => a - b);
}

export type PlanIssue = 'needDay' | 'duplicate' | 'slot';

/** Problems per row index (empty = savable). */
export function planIssues(items: readonly PlanItem[], slots: PlanSlots): Record<number, PlanIssue> {
  const out: Record<number, PlanIssue> = {};
  const seen = new Set<string>();
  items.forEach((item, i) => {
    if (!(slots[item.type] ?? []).includes(item.slot)) out[i] = 'slot';
    else if (seen.has(key(item))) out[i] = 'duplicate';
    else if (!item.days.length) out[i] = 'needDay';
    seen.add(key(item));
  });
  return out;
}
