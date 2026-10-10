import type { PainAnalgesicEffect, PainDay, PainDayInput } from '@/entities/condition';
import type { LogTaxonomy } from '@/entities/health-log';

/** What the diary form edits (one day). */
export interface PainDraft {
  score: number | null;
  locations: string[];
  types: string[];
  associated: string[];
  missedActivity: boolean;
  analgesic: string;
  analgesicTime: string;
  analgesicEffect: PainAnalgesicEffect | null;
}

export function draftFrom(day: PainDay): PainDraft {
  return {
    score: day.score,
    locations: day.score === 0 ? [] : day.locations,
    types: day.types,
    associated: day.associated,
    missedActivity: day.missedActivity ?? false,
    analgesic: day.analgesic ?? '',
    analgesicTime: day.analgesicTime ?? '',
    analgesicEffect: day.analgesicEffect,
  };
}

export function toggleCode(list: readonly string[], code: string): string[] {
  return list.includes(code) ? list.filter((c) => c !== code) : [...list, code];
}

/** The API's pairing rule, checked before sending: a score above 0 needs a location, a location needs a score. */
export function draftProblem(d: PainDraft): 'locations' | 'score' | null {
  if (d.score !== null && d.score > 0 && d.locations.length === 0) return 'locations';
  if (d.score === null && d.locations.length > 0) return 'score';
  return null;
}

export function toInput(d: PainDraft): PainDayInput {
  const analgesic = d.analgesic.trim().slice(0, 100);
  return {
    score: d.score,
    // 0 = «بدون درد»: the API clears the locations itself.
    locations: d.score === 0 ? null : d.locations,
    types: d.types,
    associated: d.associated,
    missed_activity: d.missedActivity,
    analgesic: analgesic || null,
    analgesic_time: /^\d{2}:\d{2}$/.test(d.analgesicTime) ? d.analgesicTime : null,
    analgesic_effect: d.analgesicEffect,
  };
}

export interface PainLocation {
  code: string;
  label: string;
}

/**
 * The body-map regions: the taxonomy's `pain.location` items her mode can pick, plus any location the day already
 * stores (so a saved region never disappears from the form).
 */
export function painLocations(taxonomy: LogTaxonomy | undefined, saved: readonly string[]): PainLocation[] {
  const param = taxonomy?.categories.find((c) => c.code === 'pain')?.params.find((p) => p.code === 'location');
  const mode = taxonomy?.mode ?? null;
  const options = param?.options ?? [];
  const list = options
    .filter((o) => !o.legacyOnly && (o.modes === null || mode === null || o.modes.includes(mode)))
    .map((o) => ({ code: o.value, label: o.label }));
  for (const code of saved) {
    if (!list.some((l) => l.code === code)) {
      list.push({ code, label: options.find((o) => o.value === code)?.label ?? code });
    }
  }
  return list;
}
