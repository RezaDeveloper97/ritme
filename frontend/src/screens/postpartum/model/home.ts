import type { PostpartumAlert, PostpartumRecovery, PostpartumStatus } from '@/entities/postpartum';

/** Which duration line the hero shows («۲ هفته و ۳ روز» / «۲ هفته» / «۵ روز»). */
export type DurationKey = 'weeksDays' | 'weeks' | 'days';

export function durationKey(status: Pick<PostpartumStatus, 'weeks' | 'days'>): DurationKey {
  if (status.weeks === 0) return 'days';
  return status.days === 0 ? 'weeks' : 'weeksDays';
}

/**
 * The «هفته N بعد از زایمان» chip: completed weeks (QUESTIONS #97 — `week` is
 * 1-based, the copy says «هفته ۲» on day 17). The first week reads as week 1.
 */
export function headlineWeek(status: Pick<PostpartumStatus, 'weeks'>): number {
  return Math.max(1, status.weeks);
}

/** The six-week postpartum visit, suggested while it is still ahead (day < 42). */
export function sixWeekVisitDue(status: Pick<PostpartumStatus, 'daysSinceBirth' | 'puerperiumDays'>): boolean {
  return status.daysSinceBirth < status.puerperiumDays;
}

export interface TileSummary {
  /** i18n keys / values under `postpartum.home.tiles`; null = not logged yet. */
  bleeding: { amount: string; color: string | null } | null;
  feeds: number | null;
  sleep: number | null;
}

export function tileSummary(today: PostpartumRecovery | null): TileSummary {
  return {
    bleeding: today?.lochiaAmount ? { amount: today.lochiaAmount, color: today.lochiaColor } : null,
    feeds: today?.feedsCount ?? null,
    sleep: today?.sleepHours ?? null,
  };
}

/**
 * Alerts shown on the home: today's recovery alerts (heavy bleeding, large
 * clots) first, then the overview's (check-in due …), de-duplicated by key.
 */
export function homeAlerts(overview: PostpartumAlert[], today: PostpartumRecovery | null): PostpartumAlert[] {
  const seen = new Set<string>();
  const out: PostpartumAlert[] = [];
  for (const a of [...(today?.alerts ?? []), ...overview]) {
    if (seen.has(a.key)) continue;
    seen.add(a.key);
    out.push(a);
  }
  return out;
}

/** Split `Y-m-d H:i:s` (Tehran wall clock) into the API date and `H:i`. */
export function splitScheduled(scheduledAt: string | null): { date: string; time: string } | null {
  const m = scheduledAt?.match(/^(\d{4}-\d{2}-\d{2})[ T](\d{2}:\d{2})/);
  return m ? { date: m[1], time: m[2] } : null;
}
