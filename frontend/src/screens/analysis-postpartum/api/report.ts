'use client';

import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { postpartumKeys } from '@/entities/postpartum';
import { type ApiEnvelope, apiClient, getApiErrorCode, getApiErrorStatus } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { fetchEpdsHistory, type EpdsHistory, type EpdsPoint } from './epds';

/*
 * GET /analysis/postpartum (T-M10-27, Go only): every card of An_Hub_Post
 * computed server-side, each `{plus, locked, ready, data, days_needed}`.
 * On a 404 (Laravel has no such route) the hub falls back to the EPDS-only
 * read it had before; a 409 `not_postpartum` is the inactive state.
 * Never log any of it (CLAUDE.md §11).
 */

export interface HubCard<T> {
  plus: boolean;
  locked: boolean;
  ready: boolean;
  /** null when locked (nothing computed). */
  data: T | null;
  daysNeeded: number | null;
}

export interface EpdsCardData {
  trendReady: boolean;
  /** Totals and bands only, newest first — never the answers (D-54). */
  history: EpdsHistory;
}

export interface BleedingCell {
  date: string;
  /** 0 none … 4 heavy; null = not logged. */
  level: number | null;
  code: string | null;
}

export interface BleedingCardData {
  cells: BleedingCell[];
  logged: number;
  weeks: number;
  trend: 'decreasing' | 'steady' | 'increasing' | null;
  largeClots: boolean;
  /** Today raised a bleeding red flag: point to the urgent path of /postpartum. */
  alertToday: boolean;
}

export interface FeedsCardData {
  source: 'baby_log' | 'recovery';
  childId: number | null;
  logged: number;
  feedsPerDay: number | null;
  leftPercent: number | null;
  rightPercent: number | null;
  days: { date: string; count: number | null }[];
}

export interface SleepCardData {
  motherHours: number | null;
  babySeconds: number | null;
  babyName: string | null;
}

export interface Percentile {
  percentile: number | null;
  inBand: boolean | null;
}

export interface GrowthCardData {
  available: boolean;
  child: { id: number; name: string } | null;
  latest: { weight: Percentile | null; length: Percentile | null; head: Percentile | null } | null;
  /** The T-M10-22 crossing hints, already localized by the server. */
  crossings: string[];
}

export interface WeightCardData {
  points: { date: string; kg: number }[];
  deltaSinceFirst: number | null;
  deltaVsPre: number | null;
  fromWeek: number | null;
}

export interface HubCards {
  epds: HubCard<EpdsCardData>;
  bleeding: HubCard<BleedingCardData>;
  breastfeeding: HubCard<FeedsCardData>;
  sleep: HubCard<SleepCardData>;
  growth: HubCard<GrowthCardData>;
  weight: HubCard<WeightCardData>;
}

export type HubReport =
  | { kind: 'report'; week: number; cards: HubCards }
  /** Laravel (404): the EPDS card from GET /postpartum/epds only. */
  | { kind: 'legacy'; epds: EpdsHistory }
  | { kind: 'inactive' };

const num = z.number().nullable().catch(null);
const int = z.number().int().nullable().catch(null);

function card<T extends z.ZodType>(data: T) {
  return z
    .object({
      plus: z.boolean().catch(false),
      locked: z.boolean().catch(false),
      ready: z.boolean().catch(false),
      data: data.nullable().catch(null),
      days_needed: int,
    })
    .transform((c0) => { const c = c0 as { plus: boolean; locked: boolean; ready: boolean; data: unknown; days_needed: number | null }; return ({ plus: c.plus, locked: c.locked, ready: c.ready, data: c.data as z.output<T> | null, daysNeeded: c.days_needed }); });
}

const epdsPoint = z
  .object({
    kind: z.enum(['short', 'full']),
    taken_on: z.string(),
    week: int,
    total: z.number().int(),
    max: z.number().int(),
    urgent: z.boolean().catch(false),
  })
  .transform((d): EpdsPoint => ({ kind: d.kind, takenOn: d.taken_on, week: d.week, total: d.total, max: d.max, urgent: d.urgent }));

const epdsData = z
  .object({
    trend: z.object({ ready: z.boolean().catch(false) }).catch({ ready: false }),
    points: z.array(epdsPoint).catch([]),
    thresholds: z
      .object({ short_elevated: z.number(), full_possible: z.number(), full_likely: z.number() })
      .catch({ short_elevated: 6, full_possible: 10, full_likely: 13 }),
  })
  .transform(
    (d): EpdsCardData => ({
      trendReady: d.trend.ready,
      history: {
        // The API sends the plotted checks oldest first; EpdsHistory is newest first.
        checks: d.points.slice().reverse(),
        thresholds: {
          shortElevated: d.thresholds.short_elevated,
          fullPossible: d.thresholds.full_possible,
          fullLikely: d.thresholds.full_likely,
        },
      },
    }),
  );

const bleedingData = z
  .object({
    cells: z.array(z.object({ date: z.string(), level: int, code: z.string().nullable().catch(null) })).catch([]),
    logged_days: z.number().int().catch(0),
    weeks: z.number().int().catch(1),
    trend: z.enum(['decreasing', 'steady', 'increasing']).nullable().catch(null),
    large_clots: z.boolean().catch(false),
    alert_today: z.boolean().catch(false),
  })
  .transform(
    (d): BleedingCardData => ({
      cells: d.cells,
      logged: d.logged_days,
      weeks: d.weeks,
      trend: d.trend,
      largeClots: d.large_clots,
      alertToday: d.alert_today,
    }),
  );

const feedsData = z
  .object({
    source: z.enum(['baby_log', 'recovery']).catch('recovery'),
    child_id: int,
    logged_days: z.number().int().catch(0),
    feeds_per_day: num,
    left_percent: num,
    right_percent: num,
    days: z.array(z.object({ date: z.string(), count: int })).catch([]),
  })
  .transform(
    (d): FeedsCardData => ({
      source: d.source,
      childId: d.child_id,
      logged: d.logged_days,
      feedsPerDay: d.feeds_per_day,
      leftPercent: d.left_percent,
      rightPercent: d.right_percent,
      days: d.days,
    }),
  );

const sleepData = z
  .object({
    mother: z.object({ avg_hours: num }).catch({ avg_hours: null }),
    baby: z.object({ avg_seconds: num, name: z.string().nullable().catch(null) }).catch({ avg_seconds: null, name: null }),
  })
  .transform((d): SleepCardData => ({ motherHours: d.mother.avg_hours, babySeconds: d.baby.avg_seconds, babyName: d.baby.name }));

const percentile = z
  .object({ percentile: num, in_band: z.boolean().nullable().catch(null) })
  .transform((p): Percentile => ({ percentile: p.percentile, inBand: p.in_band }))
  .nullable()
  .catch(null);

const growthData = z
  .object({
    available: z.boolean().catch(true),
    child: z.object({ id: z.number().int(), name: z.string() }).nullable().catch(null),
    latest: z.object({ weight: percentile, length: percentile, head: percentile }).nullable().catch(null),
    crossings: z
      .array(z.unknown())
      .catch([])
      .transform((list) =>
        list.flatMap((raw) => {
          const parsed = z.object({ hint: z.object({ label: z.string() }) }).safeParse(raw);
          return parsed.success ? [parsed.data.hint.label] : [];
        }),
      ),
  })
  .transform((d): GrowthCardData => d);

const weightData = z
  .object({
    points: z.array(z.object({ date: z.string(), kg: z.number() })).catch([]),
    delta_since_first: num,
    delta_vs_pre_pregnancy: num,
    from_week: int,
  })
  .transform(
    (d): WeightCardData => ({
      points: d.points,
      deltaSinceFirst: d.delta_since_first,
      deltaVsPre: d.delta_vs_pre_pregnancy,
      fromWeek: d.from_week,
    }),
  );

export const hubReportSchema = z
  .object({
    week: z.number().int().catch(1),
    cards: z.object({
      epds: card(epdsData),
      bleeding: card(bleedingData),
      breastfeeding: card(feedsData),
      sleep: card(sleepData),
      growth: card(growthData),
      weight: card(weightData),
    }),
  })
  .transform((d): HubReport => ({ kind: 'report', week: d.week, cards: d.cards }));

export async function fetchHubReport(): Promise<HubReport> {
  try {
    const { data } = await apiClient.get<ApiEnvelope<unknown>>('/analysis/postpartum');
    return hubReportSchema.parse(data.data);
  } catch (error) {
    const status = getApiErrorStatus(error);
    if (status === 409 && getApiErrorCode(error) === 'not_postpartum') return { kind: 'inactive' };
    if (status !== 404) throw error;
  }
  return { kind: 'legacy', epds: await fetchEpdsHistory() };
}

/**
 * The hub report. Keyed under `postpartumKeys.overview()`, which the EPDS
 * submit and the recovery save already invalidate.
 */
export function useHubReport() {
  return useQuery({
    queryKey: [...postpartumKeys.overview(), 'analysis-hub'] as const,
    queryFn: fetchHubReport,
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: 1,
  });
}
