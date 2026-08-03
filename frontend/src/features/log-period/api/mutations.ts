'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { articleKeys } from '@/entities/article';
import { cycleKeys } from '@/entities/cycle';
import { messageKeys } from '@/entities/message';
import { userKeys } from '@/entities/user';
import { type ApiEnvelope, apiClient } from '@/shared/api';
import { addDays, fromApiDate, today, toApiDate } from '@/shared/lib/date';
import { isAuthenticated } from '@/shared/session';

/**
 * Query key for the period logging state. Kept OUTSIDE `cycleKeys.all` so the
 * broad cycle invalidation can't race a refetch on top of the authoritative
 * value we write from the start/end response (see below).
 */
const periodStatusKey = ['period-status'] as const;

/** Query key for the logged-period history, outside `cycleKeys.all` like status. */
const periodHistoryKey = ['period-history'] as const;

const periodStatusSchema = z.object({
  active: z.boolean(),
  period_start_date: z.string().nullable().optional(),
  period_end_date: z.string().nullable().optional(),
});

export type PeriodStatus = z.infer<typeof periodStatusSchema>;

/**
 * GET /cycle/period/status — whether the user currently has an ongoing period,
 * so the button can offer "Start" vs "End". Disabled until authenticated so it
 * never fires on public screens.
 */
export function usePeriodStatus() {
  return useQuery({
    queryKey: periodStatusKey,
    queryFn: async (): Promise<PeriodStatus> => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/cycle/period/status');
      return periodStatusSchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: false,
  });
}

/**
 * Invalidate every cache that reflects the cycle after a period is logged
 * (calculations, the daily message, the profile) so the UI refetches the fresh
 * state. The period-status cache is set authoritatively by the caller instead.
 */
function useInvalidateCycle() {
  const queryClient = useQueryClient();
  return () => {
    void queryClient.invalidateQueries({ queryKey: cycleKeys.all });
    void queryClient.invalidateQueries({ queryKey: messageKeys.all });
    void queryClient.invalidateQueries({ queryKey: userKeys.all });
    // The logged-period list changed too — without this the calendar keeps a
    // stale history and re-offers "start period" right after one was logged.
    void queryClient.invalidateQueries({ queryKey: periodHistoryKey });
    // Home articles are chosen by the phase the user is in, which the new
    // period just moved — drop them so the row matches the fresh cycle.
    void queryClient.invalidateQueries({ queryKey: articleKeys.all });
  };
}

/** Extract the period status the start/end endpoints echo back in their response. */
async function postPeriod(path: string, date: string): Promise<PeriodStatus> {
  const { data } = await apiClient.post<ApiEnvelope<unknown>>(path, { date });
  return periodStatusSchema.parse(data.data);
}

/**
 * POST /cycle/period/start — record that the user's period started on `date`
 * (defaults to today). This re-anchors the cycle on the backend; the response
 * carries the new period status, which we write straight into the cache so the
 * button flips to "End period" immediately — without depending on a refetch.
 *
 * Sensitive health data (§11): the period date is sent to the API and never
 * logged. `date` crosses the boundary as Gregorian `YYYY-MM-DD`, produced only
 * via `@/shared/lib/date` (§7).
 */
export function useStartPeriod() {
  const queryClient = useQueryClient();
  const invalidate = useInvalidateCycle();
  return useMutation<PeriodStatus, unknown, { date?: string } | void>({
    mutationFn: (input) => postPeriod('/cycle/period/start', (input && input.date) || toApiDate(today())),
    onSuccess: (status) => {
      queryClient.setQueryData(periodStatusKey, status);
      invalidate();
    },
  });
}

/**
 * Log a whole bleeding range in one action (the calendar's "mark period days"
 * flow). `start` re-anchors the cycle on the backend — which cascades every
 * future prediction — and `end` (when the range is finished) closes it. Both
 * dates cross the boundary as Gregorian `YYYY-MM-DD` from `@/shared/lib/date`
 * (§7) and are never logged (§11). Runs start→end sequentially because the end
 * endpoint operates on the period the start call just anchored.
 */
export function useLogPeriodRange() {
  const queryClient = useQueryClient();
  const invalidate = useInvalidateCycle();
  return useMutation<PeriodStatus, unknown, { start: string; end?: string }>({
    mutationFn: async ({ start, end }) => {
      let status = await postPeriod('/cycle/period/start', start);
      if (end) status = await postPeriod('/cycle/period/end', end);
      return status;
    },
    onSuccess: (status) => {
      queryClient.setQueryData(periodStatusKey, status);
      invalidate();
    },
  });
}

const loggedPeriodSchema = z.object({
  id: z.number(),
  period_start_date: z.string(),
  period_end_date: z.string().nullable(),
});

export type LoggedPeriod = z.infer<typeof loggedPeriodSchema>;

/**
 * GET /cycle/period/history — every period the user actually logged (as opposed
 * to predictions), so the calendar can offer "edit this period" on its days.
 */
export function usePeriodHistory() {
  return useQuery({
    queryKey: periodHistoryKey,
    queryFn: async (): Promise<LoggedPeriod[]> => {
      const { data } = await apiClient.get<ApiEnvelope<{ periods: unknown }>>('/cycle/period/history');
      return z.array(loggedPeriodSchema).parse(data.data?.periods ?? []);
    },
    enabled: isAuthenticated(),
    staleTime: 60_000,
  });
}

/** Invalidate the caches a period edit/delete touches (history + everything cyclic). */
function useInvalidateAfterEdit() {
  const queryClient = useQueryClient();
  const invalidate = useInvalidateCycle();
  return () => {
    void queryClient.invalidateQueries({ queryKey: periodHistoryKey });
    void queryClient.invalidateQueries({ queryKey: periodStatusKey });
    invalidate();
  };
}

/**
 * PUT /cycle/period/{id} — move a logged period's start/end. The backend
 * re-anchors the cycle and regenerates all predictions. Dates are Gregorian
 * `YYYY-MM-DD` from `@/shared/lib/date` (§7) and never logged (§11).
 */
export function useUpdatePeriod() {
  const invalidateAll = useInvalidateAfterEdit();
  return useMutation<void, unknown, { id: number; start: string; end?: string | null }>({
    mutationFn: async ({ id, start, end }) => {
      await apiClient.put(`/cycle/period/${id}`, { start_date: start, end_date: end ?? null });
    },
    onSuccess: invalidateAll,
  });
}

/**
 * DELETE /cycle/period/{id} — remove a logged period entirely. The backend
 * re-anchors on the most recent remaining period (or clears the anchor).
 */
export function useDeletePeriod() {
  const invalidateAll = useInvalidateAfterEdit();
  return useMutation<void, unknown, { id: number }>({
    mutationFn: async ({ id }) => {
      await apiClient.delete(`/cycle/period/${id}`);
    },
    onSuccess: invalidateAll,
  });
}

/** A contiguous run of bleeding days the toggle editor wants persisted. */
export interface PeriodSegment {
  start: string;
  end: string;
}

/**
 * Reconcile the toggle editor's selection against the user's logged history in
 * one action. The selection arrives already grouped into contiguous `segments`
 * (each a single period); this diffs them against `existing` periods and issues
 * the minimal set of create / update / delete calls:
 *
 *  - a segment overlapping no logged period → create it,
 *  - a segment overlapping one → move that period onto it (only if it changed),
 *  - a segment overlapping several → keep the first, delete the rest (a merge),
 *  - a logged period no segment covers → delete it.
 *
 * Creates go through `POST /cycle/period` (start + end in one call) rather than
 * start-then-end, so a range can never be closed against a different period.
 *
 * An open (ongoing) period only stores its start, but reads as a full bleed of
 * `periodDuration` days everywhere else — so it's matched over that same span
 * (clamped to today) and the editor's pre-fill lines up with the calendar. Every
 * selected day is persisted: a multi-day segment records its real end (even when
 * that's today), while a lone "today" stays ongoing. A period left exactly as it
 * was is skipped, keeping an untouched open period ongoing. Deletes run first to
 * free the overlap guard, then updates, then creates. Every date is Gregorian
 * `YYYY-MM-DD` from `@/shared/lib/date` (§7) and never logged (§11).
 */
export function useReconcilePeriods() {
  const invalidateAll = useInvalidateAfterEdit();
  return useMutation<void, unknown, { segments: PeriodSegment[]; existing: LoggedPeriod[]; periodDuration: number }>({
    mutationFn: async ({ segments, existing, periodDuration }) => {
      const todayIso = toApiDate(today());
      const clampToday = (iso: string) => (iso > todayIso ? todayIso : iso);
      // How far a logged period visibly reaches: its recorded end, or (while open)
      // the profile's usual bleed length, never past today. Mirrors the calendar.
      const effEnd = (p: LoggedPeriod) =>
        p.period_end_date ??
        clampToday(toApiDate(addDays(fromApiDate(p.period_start_date), Math.max(0, periodDuration - 1))));
      const overlaps = (seg: PeriodSegment, s: string, e: string) => seg.start <= e && s <= seg.end;
      // A lone "today" is still ongoing (no end); anything else records its last day.
      const endForSegment = (seg: PeriodSegment) =>
        seg.start === todayIso && seg.end === todayIso ? null : seg.end;

      const claimed = new Set<number>();
      const deletes: number[] = [];
      const updates: { id: number; start: string; end: string | null }[] = [];
      const creates: PeriodSegment[] = [];

      for (const seg of segments) {
        const hits = existing.filter((p) => overlaps(seg, p.period_start_date, effEnd(p)));
        if (hits.length === 0) {
          creates.push(seg);
          continue;
        }
        const [keep, ...rest] = hits;
        claimed.add(keep!.id);
        for (const r of rest) {
          claimed.add(r.id);
          deletes.push(r.id);
        }
        // Untouched (range unchanged) → leave the record alone, preserving an
        // open period's ongoing status. Otherwise move it onto the segment.
        const unchanged = keep!.period_start_date === seg.start && effEnd(keep!) === seg.end;
        if (!unchanged) {
          updates.push({ id: keep!.id, start: seg.start, end: endForSegment(seg) });
        }
      }
      for (const p of existing) if (!claimed.has(p.id)) deletes.push(p.id);

      for (const id of [...new Set(deletes)]) {
        await apiClient.delete(`/cycle/period/${id}`);
      }
      for (const u of updates) {
        await apiClient.put(`/cycle/period/${u.id}`, { start_date: u.start, end_date: u.end });
      }
      for (const c of creates) {
        // One call per range: `/period/start` + `/period/end` would close whichever
        // period the backend considers *ongoing* — with several ranges in flight (or
        // an untouched open period) that is not the one just created, and the end
        // date then lands before its start (422).
        await apiClient.post('/cycle/period', { start_date: c.start, end_date: endForSegment(c) });
      }
    },
    onSuccess: invalidateAll,
  });
}

/**
 * POST /cycle/period/end — close the ongoing period with an end date (defaults
 * to today). Writes the returned status into the cache so the button flips back
 * to "Start period" at once.
 */
export function useEndPeriod() {
  const queryClient = useQueryClient();
  const invalidate = useInvalidateCycle();
  return useMutation<PeriodStatus, unknown, { date?: string } | void>({
    mutationFn: (input) => postPeriod('/cycle/period/end', (input && input.date) || toApiDate(today())),
    onSuccess: (status) => {
      queryClient.setQueryData(periodStatusKey, status);
      invalidate();
    },
  });
}
