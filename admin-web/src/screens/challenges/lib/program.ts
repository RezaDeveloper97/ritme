import type { Translations } from '@/shared/ui';

import type { Challenge } from '../api/challenges';

/** Life modes and cycle main phases an eligibility rule can name (the API validates the same lists). */
export const MODES = ['cycle', 'ttc', 'pregnancy', 'postpartum', 'menopause', 'teen'] as const;
export const PHASES = ['menstrual', 'follicular', 'fertile', 'luteal', 'period_expected', 'unknown'] as const;
export const START_RULES = ['anytime', 'menstrual', 'fixed_date'] as const;

export interface TaskDraft {
  /** Stable React key (the server id when saved, else a counter). */
  key: string;
  id: number | null;
  dayFrom: string;
  dayTo: string;
  title: Translations;
}

export interface ProgramDraft {
  enabled: boolean;
  duration: string;
  startRule: string;
  startsOn: string;
  reminder: string;
  modesInclude: string[];
  modesExclude: string[];
  phasesInclude: string[];
  phasesExclude: string[];
  badgeCode: string;
  badgeTitle: Translations;
  plusDays: string;
  minDays: string;
  tasks: TaskDraft[];
}

let counter = 0;
export const newTask = (dayFrom = '1', dayTo = ''): TaskDraft => ({ key: `new-${counter++}`, id: null, dayFrom, dayTo, title: {} });

export function toDraft(row: Challenge | null): ProgramDraft {
  const p = row?.program ?? null;
  return {
    enabled: p !== null,
    duration: p ? String(p.duration_days) : '7',
    startRule: p?.start_rule ?? 'anytime',
    startsOn: p?.starts_on ?? '',
    reminder: p?.reminder_time ?? '',
    modesInclude: p?.modes_include ?? [],
    modesExclude: p?.modes_exclude ?? [],
    phasesInclude: p?.phases_include ?? [],
    phasesExclude: p?.phases_exclude ?? [],
    badgeCode: p?.reward_badge_code ?? '',
    badgeTitle: p?.reward_badge_title ?? {},
    plusDays: String(p?.reward_plus_days ?? 0),
    minDays: p?.completion_min_days != null ? String(p.completion_min_days) : '',
    tasks: (row?.tasks ?? []).map((t) => ({ key: String(t.id), id: t.id, dayFrom: String(t.day_from), dayTo: String(t.day_to), title: t.title })),
  };
}

const int = (s: string): number | null => (s.trim() === '' || Number.isNaN(Number(s)) ? null : Math.trunc(Number(s)));

/** The v2 part of the write body — empty when the challenge stays a legacy one-day pick. */
export function toPayload(d: ProgramDraft): Record<string, unknown> {
  if (!d.enabled) return {};
  return {
    duration_days: int(d.duration),
    start_rule: d.startRule,
    starts_on: d.startRule === 'fixed_date' && d.startsOn ? d.startsOn : null,
    reminder_time: d.reminder || null,
    modes_include: d.modesInclude,
    modes_exclude: d.modesExclude,
    phases_include: d.phasesInclude,
    phases_exclude: d.phasesExclude,
    reward_badge_code: d.badgeCode.trim() || null,
    reward_badge_title: d.badgeTitle,
    reward_plus_days: int(d.plusDays) ?? 0,
    completion_min_days: int(d.minDays),
    tasks: d.tasks.map((t, i) => ({
      ...(t.id !== null ? { id: t.id } : {}),
      day_from: int(t.dayFrom),
      day_to: int(t.dayTo) ?? int(t.dayFrom),
      title: t.title,
      sort_order: i,
    })),
  };
}

/** Day numbers 1..duration, for the per-day task overview. */
export function daysOf(duration: string): number[] {
  const n = int(duration);
  return n !== null && n >= 1 && n <= 366 ? Array.from({ length: n }, (_, i) => i + 1) : [];
}

/** The tasks that apply to a day (1-based). */
export function tasksOnDay(tasks: readonly TaskDraft[], day: number): TaskDraft[] {
  return tasks.filter((t) => {
    const from = int(t.dayFrom);
    const to = int(t.dayTo) ?? from;
    return from !== null && to !== null && from <= day && day <= to;
  });
}
