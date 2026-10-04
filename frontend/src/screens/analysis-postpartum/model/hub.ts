import type { LogDay } from '@/entities/health-log';

import type { EpdsHistory, EpdsPoint } from '../api/epds';

/*
 * The postpartum analysis hub (An_Hub_Post, B-N5-07) is computed client-side
 * from endpoints that already exist: EPDS history, the taxonomy v2 day range
 * (bleeding.lochia_amount / clot_size, sleep.hours, measurements.weight) and
 * the baby-log summary. Descriptive only — no score is interpreted beyond the
 * cut-offs the API echoes. Pure functions over `Y-m-d` strings, no locale.
 */

// ── dates ──────────────────────────────────────────────────────
const DAY = 86_400_000;
const utc = (d: string) => {
  const [y, m, dd] = d.split('-').map(Number);
  return Date.UTC(y, m - 1, dd);
};
export const shiftDate = (d: string, days: number) => new Date(utc(d) + days * DAY).toISOString().slice(0, 10);
export const daysBetween = (a: string, b: string) => Math.round((utc(a) - utc(b)) / DAY);

/** Week after the birth that `date` falls in (day 0–6 = week 1). */
export function weekAfterBirth(date: string, birth: string): number {
  return Math.floor(Math.max(0, daysBetween(date, birth)) / 7) + 1;
}

/** The `/logs/days` range the hub reads: from the birth (at most 366 days back) to today. */
export function hubRange(today: string, birth: string | null): { from: string; to: string } {
  const earliest = shiftDate(today, -365);
  const from = birth && birth > earliest ? (birth > today ? today : birth) : earliest;
  return { from, to: today };
}

// ── EPDS ───────────────────────────────────────────────────────
export interface EpdsTrend {
  kind: 'short' | 'full';
  /** Oldest first. */
  points: EpdsPoint[];
  max: number;
  /** Full: «possible» (10) and «likely» (13) cut-offs; short: the single «elevated» one (6). */
  lower: number;
  upper: number | null;
  latest: EpdsPoint;
  /** The highest earlier score at or above `lower`, when the latest is below it. */
  peak: EpdsPoint | null;
  status: 'urgent' | 'high' | 'down' | 'low';
}

/**
 * The trend card plots one questionnaire: the full EPDS when there is any
 * (its 10 / 13 bands are the ones the artboard draws), else the short one.
 */
export function epdsTrend(history: EpdsHistory): EpdsTrend | null {
  const full = history.checks.filter((c) => c.kind === 'full');
  const kind = full.length ? 'full' : 'short';
  const points = (full.length ? full : history.checks.filter((c) => c.kind === 'short')).slice().reverse();
  if (!points.length) return null;
  const th = history.thresholds;
  const lower = kind === 'full' ? th.fullPossible : th.shortElevated;
  const upper = kind === 'full' ? th.fullLikely : null;
  const latest = points[points.length - 1];
  const earlier = points.slice(0, -1).filter((p) => p.total >= lower);
  const peak = latest.total < lower && earlier.length ? earlier.reduce((a, b) => (b.total > a.total ? b : a)) : null;
  // Any urgent check in the history (self-harm item, or ≥ likely) keeps the card's call-your-doctor line.
  const urgent = latest.urgent || (upper !== null && latest.total >= upper);
  const status = urgent ? 'urgent' : latest.total >= lower ? 'high' : peak ? 'down' : 'low';
  return { kind, points, max: latest.max, lower, upper, latest, peak, status };
}

// ── bleeding (lochia) ──────────────────────────────────────────
const LOCHIA_LEVEL: Record<string, number> = { none: 0, spotting: 1, light: 2, medium: 3, heavy: 4 };

export interface BleedingCell {
  date: string;
  /** 0 none … 4 heavy; null = not logged. */
  level: number | null;
  code: string | null;
}

export interface BleedingTrend {
  cells: BleedingCell[];
  logged: number;
  trend: 'decreasing' | 'steady' | 'increasing' | null;
  largeClots: boolean;
  weeks: number;
}

function valueOf(day: LogDay | undefined, category: string, param: string): unknown {
  return day?.categories[category]?.[param];
}

/**
 * The last `span` days (oldest first, never before the birth) of
 * `bleeding.lochia_amount`, the direction from the first third to the last
 * third of the logged days, and whether a large clot was logged.
 */
export function bleedingTrend(days: readonly LogDay[], today: string, birth: string | null, span = 21): BleedingTrend {
  const byDate = new Map(days.map((d) => [d.date, d]));
  const sinceBirth = birth ? daysBetween(today, birth) + 1 : span;
  const n = Math.max(1, Math.min(span, sinceBirth));
  const cells: BleedingCell[] = [];
  let largeClots = false;
  for (let i = n - 1; i >= 0; i--) {
    const date = shiftDate(today, -i);
    const day = byDate.get(date);
    const raw = valueOf(day, 'bleeding', 'lochia_amount');
    const code = typeof raw === 'string' && raw in LOCHIA_LEVEL ? raw : null;
    cells.push({ date, level: code ? LOCHIA_LEVEL[code] : null, code });
    if (valueOf(day, 'bleeding', 'clot_size') === 'large') largeClots = true;
  }
  const levels = cells.flatMap((c) => (c.level === null ? [] : [c.level]));
  let trend: BleedingTrend['trend'] = null;
  if (levels.length >= 4) {
    const third = Math.max(1, Math.floor(levels.length / 3));
    const mean = (xs: number[]) => xs.reduce((a, b) => a + b, 0) / xs.length;
    const delta = mean(levels.slice(-third)) - mean(levels.slice(0, third));
    trend = delta <= -0.5 ? 'decreasing' : delta >= 0.5 ? 'increasing' : 'steady';
  }
  return { cells, logged: levels.length, trend, largeClots, weeks: Math.max(1, Math.round(n / 7)) };
}

// ── mother's sleep & weight ────────────────────────────────────
function numberOf(day: LogDay | undefined, category: string, param: string): number | null {
  const v = valueOf(day, category, param);
  return typeof v === 'number' && Number.isFinite(v) ? v : null;
}

/** Mean of the logged `sleep.hours` over the last `span` days (today included), null when none. */
export function motherSleepAverage(days: readonly LogDay[], today: string, span = 7): number | null {
  const from = shiftDate(today, -(span - 1));
  const hours = days.filter((d) => d.date >= from && d.date <= today).flatMap((d) => {
    const h = numberOf(d, 'sleep', 'hours');
    return h === null ? [] : [h];
  });
  return hours.length ? hours.reduce((a, b) => a + b, 0) / hours.length : null;
}

export interface WeightTrend {
  /** Oldest first. */
  points: { date: string; kg: number }[];
  first: { date: string; kg: number };
  latest: { date: string; kg: number };
  /** latest − first, kg. */
  delta: number;
  /** Week after the birth of the first weigh-in («از هفته ۱»). */
  fromWeek: number;
}

/** `measurements.weight` since the birth; null without any weigh-in. */
export function weightTrend(days: readonly LogDay[], birth: string | null): WeightTrend | null {
  const points = days
    .filter((d) => !birth || d.date >= birth)
    .flatMap((d) => {
      const kg = numberOf(d, 'measurements', 'weight');
      return kg === null ? [] : [{ date: d.date, kg }];
    })
    .sort((a, b) => (a.date < b.date ? -1 : 1));
  if (!points.length) return null;
  const first = points[0];
  const latest = points[points.length - 1];
  return {
    points,
    first,
    latest,
    delta: Math.round((latest.kg - first.kg) * 10) / 10,
    fromWeek: birth ? weekAfterBirth(first.date, birth) : 1,
  };
}

// ── the child ──────────────────────────────────────────────────
/**
 * The hub's baby: the own child born closest to the postpartum birth date
 * (twins: the first listed), else the first own child, else none.
 */
export function hubChildId(
  children: ReadonlyArray<{ id: number; role: string; birthDate: string }>,
  birth: string | null,
): number | null {
  const own = children.filter((c) => c.role === 'owner');
  if (!own.length) return null;
  if (!birth) return own[0].id;
  return own.reduce((best, c) =>
    Math.abs(daysBetween(c.birthDate, birth)) < Math.abs(daysBetween(best.birthDate, birth)) ? c : best,
  ).id;
}

/** EPDS chart geometry: x per point (oldest left), y on a 0..max scale with the cut-off bands. */
export const EPDS_W = 320;
export function epdsGeometry(trend: EpdsTrend, height: number) {
  const padX = 24;
  const padTop = 8;
  const padBottom = 8;
  // The scale hugs the scores (the artboard puts 10 / 13 mid-chart), not the questionnaire's 0–30.
  const top = Math.min(trend.max, Math.max(...trend.points.map((p) => p.total), trend.upper ?? trend.lower) + 5);
  const bottom = Math.max(0, Math.min(...trend.points.map((p) => p.total)) - 4);
  const y = (v: number) => padTop + (height - padTop - padBottom) * (1 - (v - bottom) / (top - bottom || 1));
  const n = trend.points.length;
  const x = (i: number) => (n <= 1 ? EPDS_W / 2 : padX + ((EPDS_W - padX - 8) * i) / (n - 1));
  const dots = trend.points.map((p, i) => ({
    x: Math.round(x(i) * 10) / 10,
    y: Math.round(y(p.total) * 10) / 10,
    high: p.total >= trend.lower,
  }));
  const line = dots.map((d, i) => `${i ? 'L' : 'M'}${d.x} ${d.y}`).join('');
  return {
    dots,
    line,
    lowerY: y(trend.lower),
    upperY: trend.upper !== null ? y(trend.upper) : null,
    topY: padTop,
    padX,
  };
}
