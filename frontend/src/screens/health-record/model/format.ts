import type { HealthRecord, PregnancyEntry, RecordMedication } from '@/entities/health-record';

/** How a medication's schedule reads: every weekday → daily, else the number of days a week. */
export function medicationSchedule(m: Pick<RecordMedication, 'weekdays'>): { kind: 'daily' } | { kind: 'weekly'; days: number } {
  const days = new Set(m.weekdays).size;
  return days === 0 || days >= 7 ? { kind: 'daily' } : { kind: 'weekly', days };
}

/** The «نظم» value key of a regularity code (falls back to not enough data). */
export function regularityKey(code: string): 'regular' | 'irregular' | 'not_enough_data' {
  return code === 'regular' || code === 'irregular' ? code : 'not_enough_data';
}

/** Whether the whole record has nothing in it yet (every section empty). */
export function isRecordEmpty(r: HealthRecord): boolean {
  return [r.basics, r.conditions, r.medications, r.allergies, r.cycle, r.vitals, r.pregnancies, r.checkups, r.labs].every(
    (s) => s.empty,
  );
}

/** The manual entries the pregnancies sheet can edit. */
export function manualEntries(items: readonly PregnancyEntry[]): (PregnancyEntry & { id: number })[] {
  return items.filter((e): e is PregnancyEntry & { id: number } => e.editable && e.id !== null);
}

/** The birth rows of the pregnancies card («زایمان · طبیعی · ۱۴۰۳»). */
export function births(items: readonly PregnancyEntry[]): PregnancyEntry[] {
  return items.filter((e) => e.outcome === 'vaginal' || e.outcome === 'cesarean' || e.outcome === 'birth');
}
