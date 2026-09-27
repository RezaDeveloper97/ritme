import type { IconName } from '@/shared/ui';

import {
  type DatingPreviewInput,
  type HighlightIcon,
  type HighlightTone,
  type Mood,
  type PregnancyProgressV2,
  V2_MAX_WEEK,
  V2_MIN_WEEK,
  type WeekRelation,
  type WeekTask,
} from './v2-types';

/*
 * Pure helpers for pregnancy v2 (no I/O, no React) — shared by the Today,
 * Week and Log screens (T-M7-10…12) and unit-tested in `v2.test.ts`.
 */

/** Glyph + default tile tint per admin-pickable highlight icon (Week artboard). */
const HIGHLIGHT_GLYPHS: Record<HighlightIcon, { icon: IconName; tone: HighlightTone }> = {
  hand: { icon: 'hand', tone: 'brand' },
  heart: { icon: 'heartLine', tone: 'pink' },
  eye: { icon: 'eye', tone: 'teal' },
  brain: { icon: 'brain', tone: 'brand' },
  drop: { icon: 'drop', tone: 'teal' },
  scale: { icon: 'scale', tone: 'brand' },
  sparkle: { icon: 'sparkle', tone: 'pink' },
  moon: { icon: 'moonReminder', tone: 'brand' },
  baby: { icon: 'heartLine', tone: 'pink' },
  face: { icon: 'faceGood', tone: 'teal' },
};

/** The icon for a highlight; an unknown / missing key falls back to a sparkle. */
export function highlightGlyph(icon: HighlightIcon | null): IconName {
  return icon ? HIGHLIGHT_GLYPHS[icon].icon : 'sparkle';
}

/** The default tint of a highlight icon (the admin may override it). */
export function defaultHighlightTone(icon: HighlightIcon | null): HighlightTone {
  return icon ? HIGHLIGHT_GLYPHS[icon].tone : 'brand';
}

/** The Log artboard's face for a mood (5 = best day … 1 = hardest). */
const MOOD_GLYPHS: Record<Mood, IconName> = {
  5: 'faceGreat',
  4: 'faceGood',
  3: 'faceOkay',
  2: 'faceLow',
  1: 'faceHard',
};

export function moodGlyph(mood: Mood): IconName {
  return MOOD_GLYPHS[mood];
}

/** Is `n` a week the v2 API serves (`/weeks/{n}` answers 404 otherwise)? */
export function isV2Week(n: number): boolean {
  return Number.isInteger(n) && n >= V2_MIN_WEEK && n <= V2_MAX_WEEK;
}

/** Clamp any number into the 1–42 v2 week range. */
export function clampV2Week(n: number): number {
  if (!Number.isFinite(n)) return V2_MIN_WEEK;
  return Math.min(V2_MAX_WEEK, Math.max(V2_MIN_WEEK, Math.round(n)));
}

/** Week chip state relative to the current week. */
export function weekRelation(week: number, currentWeek: number | null): WeekRelation {
  if (currentWeek == null) return 'future';
  if (week === currentWeek) return 'current';
  return week < currentWeek ? 'past' : 'future';
}

/** «۲ از ۳» — done count for the «مراقبت‌های این هفته» header. */
export function taskProgress(tasks: readonly WeekTask[]): { done: number; total: number } {
  return { done: tasks.filter((t) => t.done).length, total: tasks.length };
}

/** The done-key list after ticking / unticking `key` (order kept, no dupes). */
export function toggleDoneKey(doneKeys: readonly string[], key: string): string[] {
  return doneKeys.includes(key) ? doneKeys.filter((k) => k !== key) : [...doneKeys, key];
}

/** Apply a done-key list to a task list (used for optimistic updates). */
export function applyDoneKeys(tasks: readonly WeekTask[], doneKeys: readonly string[]): WeekTask[] {
  const done = new Set(doneKeys);
  return tasks.map((t) => ({ ...t, done: done.has(t.key) }));
}

/** Only the fields of the chosen source, so the cache key is stable. */
export function datingPreviewBody(input: DatingPreviewInput): DatingPreviewInput {
  switch (input.source) {
    case 'lmp':
      return { source: 'lmp', lmp_date: input.lmp_date };
    case 'ultrasound':
      return {
        source: 'ultrasound',
        ultrasound_date: input.ultrasound_date,
        ultrasound_weeks: input.ultrasound_weeks,
        ultrasound_days: input.ultrasound_days ?? 0,
      };
    case 'manual':
      return { source: 'manual', manual_weeks: input.manual_weeks, manual_days: input.manual_days ?? 0 };
  }
}

/** Has the chosen source everything the server needs? */
export function isDatingPreviewReady(input: DatingPreviewInput | null): input is DatingPreviewInput {
  if (!input) return false;
  switch (input.source) {
    case 'lmp':
      return !!input.lmp_date;
    case 'ultrasound':
      return !!input.ultrasound_date && input.ultrasound_weeks != null;
    case 'manual':
      return input.manual_weeks != null;
  }
}

/** One segment of the Today 40-week bar. */
export interface TrimesterFill {
  trimester: 1 | 2 | 3;
  /** 0–100: how much of this trimester is behind the user. */
  fill: number;
  /** Has the user reached this trimester yet? */
  started: boolean;
}

/**
 * Per-trimester fill of the Today progress bar. The API sends each trimester's
 * **start** on the 40-week bar (`trimesters[].percent`) and the overall
 * `percent`; a segment runs from its start to the next one's (or 100).
 */
export function trimesterFills(progress: Pick<PregnancyProgressV2, 'percent' | 'trimesters'>): TrimesterFill[] {
  const spans = [...progress.trimesters].sort((a, b) => a.percent - b.percent);
  const overall = Math.min(100, Math.max(0, progress.percent));
  return spans.map((s, i) => {
    const start = s.percent;
    const end = spans[i + 1]?.percent ?? 100;
    const width = end - start;
    const fill = width > 0 ? Math.round(Math.min(1, Math.max(0, (overall - start) / width)) * 100) : 0;
    // Overall percent is rounded, so sitting exactly on a later start still means «not yet».
    return { trimester: s.trimester, fill, started: start === 0 || overall > start };
  });
}
