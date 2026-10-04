/*
 * Baby logs (bloom B-N5-03 API, B-N5-07 screens): feeds (breast timer with
 * per-side totals, bottle, pump), sleeps and diapers per child. Mirrors
 * `BabyFeed*`, `BabySleep*`, `BabyDiaper*`, `BabyLogSummary` of
 * backend-go/api/openapi.yaml. Health data (CLAUDE.md §11): never logged.
 */

export const FEED_TYPES = ['breast', 'bottle', 'pump'] as const;
export type FeedType = (typeof FEED_TYPES)[number];

export const FEED_SIDES = ['left', 'right'] as const;
export type FeedSide = (typeof FEED_SIDES)[number];

export const DIAPER_KINDS = ['wet', 'dirty', 'both'] as const;
export type DiaperKind = (typeof DIAPER_KINDS)[number];

/** Validation mirrors of `internal/babylog` (the API decides). */
export const MAX_AMOUNT_ML = 500;
export const MAX_FEED_MINUTES = 180;
export const MAX_SLEEP_HOURS = 24;
export const SUMMARY_DAYS_MAX = 31;

/** One feed. Times are Tehran wall-clock ISO strings (`2026-09-23T14:00:00+03:30`). */
export interface BabyFeed {
  id: number;
  type: FeedType;
  startedAt: string;
  endedAt: string | null;
  isActive: boolean;
  /** The side the breast timer runs on now (null = paused, or not a breast feed). */
  activeSide: FeedSide | null;
  /** When the running side segment started. */
  sideStartedAt: string | null;
  /** The side the feed ended (or was last switched away) on. */
  lastSide: FeedSide | null;
  /** Closed side segments only — the running one is added live (`model/live.ts`). */
  leftSeconds: number;
  rightSeconds: number;
  durationSeconds: number;
  amountMl: number | null;
  note: string | null;
}

export interface FeedTotals {
  count: number;
  breast: number;
  bottle: number;
  pump: number;
  leftSeconds: number;
  rightSeconds: number;
  totalSeconds: number;
  bottleMl: number;
  pumpMl: number;
}

/** GET /children/{id}/feeds — the running feed, the last ended one, the side to offer first, the day. */
export interface FeedDay {
  date: string;
  active: BabyFeed | null;
  last: BabyFeed | null;
  nextSide: FeedSide | null;
  summary: FeedTotals;
  items: BabyFeed[];
}

export interface BabySleep {
  id: number;
  startedAt: string;
  endedAt: string | null;
  isActive: boolean;
  durationSeconds: number;
  note: string | null;
}

export interface SleepTotals {
  count: number;
  seconds: number;
  longestSeconds: number;
}

export interface SleepDay {
  date: string;
  active: BabySleep | null;
  summary: SleepTotals;
  items: BabySleep[];
}

export interface BabyDiaper {
  id: number;
  changedAt: string;
  kind: DiaperKind;
  note: string | null;
}

export interface DiaperTotals {
  count: number;
  wet: number;
  dirty: number;
  both: number;
}

export interface DiaperDay {
  date: string;
  summary: DiaperTotals;
  items: BabyDiaper[];
}

export interface BabyLogDay {
  date: string;
  feeds: FeedTotals;
  sleep: SleepTotals;
  diapers: DiaperTotals;
}

/** The child home's «امروز» card (`ChildHome.today`). */
export interface BabyToday extends BabyLogDay {
  lastFeed: BabyFeed | null;
  feedingNow: boolean;
  sleepingNow: boolean;
}

/** GET /children/{id}/baby-logs?days= (An_Hub_Post). */
export interface BabyLogSummary {
  from: string;
  to: string;
  today: BabyToday;
  /** Oldest first, today included. */
  days: BabyLogDay[];
  averages: {
    feedsPerDay: number;
    sleepSecondsPerDay: number;
    diapersPerDay: number;
    leftPercent: number | null;
    rightPercent: number | null;
  };
}

/** POST /children/{id}/feeds (an ended feed by hand). */
export interface ManualFeedInput {
  type: FeedType;
  /** `Y-m-d H:i`, Tehran wall-clock. */
  startedAt: string;
  leftMinutes?: number;
  rightMinutes?: number;
  durationMinutes?: number;
  amountMl?: number | null;
}

/** POST /children/{id}/sleeps (an ended sleep by hand). `Y-m-d H:i`, Tehran wall-clock. */
export interface ManualSleepInput {
  startedAt: string;
  endedAt: string;
}
