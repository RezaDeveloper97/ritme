'use client';

import { type QueryClient, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { childKeys } from '@/entities/child';
import { logKeys } from '@/entities/health-log';
import { postpartumKeys } from '@/entities/postpartum';
import { type ApiEnvelope, ApiError, apiClient, getApiErrorCode } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type {
  BabyDiaper,
  BabyFeed,
  BabyLogSummary,
  BabySleep,
  DiaperDay,
  DiaperKind,
  FeedDay,
  FeedSide,
  FeedType,
  ManualFeedInput,
  ManualSleepInput,
  SleepDay,
} from '../model/types';
import { babyLogKeys } from './keys';
import {
  babyLogSummarySchema,
  diaperDaySchema,
  diaperSchema,
  feedDaySchema,
  feedSchema,
  sleepDaySchema,
  sleepSchema,
} from './schema';

/*
 * `/api/v1/children/{id}/feeds|sleeps|diapers|baby-logs` (B-N5-03), Go only.
 * One running feed and one running sleep per child live on the server, so the
 * timer survives a reload and shows on the spouse's phone too. Health data
 * (CLAUDE.md §11): never log a payload or a response.
 */

const enabledFor = (id: number | null) => isAuthenticated() && id != null && id > 0;
const notFound = (error: unknown) => error instanceof ApiError && error.response?.status === 404;
const retry = (count: number, error: unknown) => !notFound(error) && count < 1;

export async function fetchFeedDay(id: number, date: string | null): Promise<FeedDay> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/children/${id}/feeds`, {
    params: date ? { date } : undefined,
  });
  return feedDaySchema.parse(data.data);
}

/** GET /children/{id}/feeds[?date] — the running feed, last feed, next side and the day's totals. */
export function useFeedDay(id: number | null, date: string | null = null) {
  return useQuery({
    queryKey: babyLogKeys.feeds(id ?? 0, date),
    queryFn: () => fetchFeedDay(id as number, date),
    enabled: enabledFor(id),
    staleTime: 15_000,
    // A running feed may be switched or stopped on the other parent's phone.
    refetchInterval: (q) => (q.state.data?.active ? 30_000 : false),
    retry,
  });
}

export async function fetchSleepDay(id: number, date: string | null): Promise<SleepDay> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/children/${id}/sleeps`, {
    params: date ? { date } : undefined,
  });
  return sleepDaySchema.parse(data.data);
}

/** GET /children/{id}/sleeps[?date] — the running sleep and the day's sleep. */
export function useSleepDay(id: number | null, date: string | null = null) {
  return useQuery({
    queryKey: babyLogKeys.sleeps(id ?? 0, date),
    queryFn: () => fetchSleepDay(id as number, date),
    enabled: enabledFor(id),
    staleTime: 15_000,
    refetchInterval: (q) => (q.state.data?.active ? 60_000 : false),
    retry,
  });
}

export async function fetchDiaperDay(id: number, date: string | null): Promise<DiaperDay> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/children/${id}/diapers`, {
    params: date ? { date } : undefined,
  });
  return diaperDaySchema.parse(data.data);
}

/** GET /children/{id}/diapers[?date] — the day's changes. */
export function useDiaperDay(id: number | null, date: string | null = null) {
  return useQuery({
    queryKey: babyLogKeys.diapers(id ?? 0, date),
    queryFn: () => fetchDiaperDay(id as number, date),
    enabled: enabledFor(id),
    staleTime: 15_000,
    retry,
  });
}

export async function fetchBabyLogSummary(id: number, days: number): Promise<BabyLogSummary> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/children/${id}/baby-logs`, { params: { days } });
  return babyLogSummarySchema.parse(data.data);
}

/** GET /children/{id}/baby-logs?days — today, the last `days` days (oldest first) and their averages. */
export function useBabyLogSummary(id: number | null, days = 7) {
  return useQuery({
    queryKey: babyLogKeys.summary(id ?? 0, days),
    queryFn: () => fetchBabyLogSummary(id as number, days),
    enabled: enabledFor(id),
    staleTime: 60_000,
    retry,
  });
}

/** After any write: the child's logs, the child home «امروز», and the mother's feeds_count (postpartum). */
function refresh(queryClient: QueryClient, id: number) {
  void queryClient.invalidateQueries({ queryKey: babyLogKeys.child(id) });
  void queryClient.invalidateQueries({ queryKey: childKeys.detail(id) });
  void queryClient.invalidateQueries({ queryKey: postpartumKeys.all });
  void queryClient.invalidateQueries({ queryKey: logKeys.days() });
}

/** Puts a running / just-ended feed into today's cached day at once (the refetch then settles the totals). */
function putFeed(queryClient: QueryClient, id: number, feed: BabyFeed) {
  queryClient.setQueryData<FeedDay>(babyLogKeys.feeds(id, null), (day) => {
    if (!day) return day;
    const items = [feed, ...day.items.filter((f) => f.id !== feed.id)];
    return { ...day, active: feed.isActive ? feed : null, last: feed.isActive ? day.last : feed, items };
  });
}

function putSleep(queryClient: QueryClient, id: number, sleep: BabySleep) {
  queryClient.setQueryData<SleepDay>(babyLogKeys.sleeps(id, null), (day) => {
    if (!day) return day;
    return { ...day, active: sleep.isActive ? sleep : null, items: [sleep, ...day.items.filter((s) => s.id !== sleep.id)] };
  });
}

export type FeedAction =
  | { kind: 'start'; type: FeedType; side?: FeedSide | null }
  | { kind: 'side'; feedId: number; side: FeedSide | null }
  | { kind: 'stop'; feedId: number; amountMl?: number | null }
  | { kind: 'discard'; feedId: number }
  | { kind: 'manual'; input: ManualFeedInput };

/** Snake-case body of a manual feed: breast sends side minutes, bottle / pump duration and ml. */
export function toManualFeedBody(input: ManualFeedInput): Record<string, unknown> {
  const body: Record<string, unknown> = { type: input.type, started_at: input.startedAt };
  if (input.type === 'breast') {
    body.left_minutes = input.leftMinutes ?? 0;
    body.right_minutes = input.rightMinutes ?? 0;
  } else {
    body.duration_minutes = input.durationMinutes ?? 0;
    body.amount_ml = input.amountMl ?? null;
  }
  return body;
}

/** Every feed write of the timer screen in one mutation (start, switch / pause side, stop, discard, manual). */
export function useFeedAction(id: number) {
  const queryClient = useQueryClient();
  return useMutation<BabyFeed | null, unknown, FeedAction>({
    mutationFn: async (action) => {
      const base = `/children/${id}/feeds`;
      switch (action.kind) {
        case 'start': {
          const body = action.type === 'breast' && action.side ? { type: action.type, side: action.side } : { type: action.type };
          const { data } = await apiClient.post<ApiEnvelope<unknown>>(`${base}/start`, body);
          return feedSchema.parse(data.data);
        }
        case 'side': {
          const { data } = await apiClient.post<ApiEnvelope<unknown>>(`${base}/${action.feedId}/side`, { side: action.side });
          return feedSchema.parse(data.data);
        }
        case 'stop': {
          const body = action.amountMl != null ? { amount_ml: action.amountMl } : {};
          const { data } = await apiClient.post<ApiEnvelope<unknown>>(`${base}/${action.feedId}/stop`, body);
          return feedSchema.parse(data.data);
        }
        case 'discard':
          await apiClient.delete(`${base}/${action.feedId}`);
          return null;
        case 'manual': {
          const { data } = await apiClient.post<ApiEnvelope<unknown>>(base, toManualFeedBody(action.input));
          return feedSchema.parse(data.data);
        }
      }
    },
    onSuccess: (feed, action) => {
      if (feed) putFeed(queryClient, id, feed);
      else if (action.kind === 'discard') {
        queryClient.setQueryData<FeedDay>(babyLogKeys.feeds(id, null), (day) =>
          day ? { ...day, active: null, items: day.items.filter((f) => f.id !== action.feedId) } : day,
        );
      }
    },
    onSettled: () => refresh(queryClient, id),
  });
}

export type SleepAction =
  | { kind: 'start' }
  | { kind: 'stop'; sleepId: number }
  | { kind: 'manual'; input: ManualSleepInput }
  | { kind: 'delete'; sleepId: number };

export function useSleepAction(id: number) {
  const queryClient = useQueryClient();
  return useMutation<BabySleep | null, unknown, SleepAction>({
    mutationFn: async (action) => {
      const base = `/children/${id}/sleeps`;
      switch (action.kind) {
        case 'start': {
          const { data } = await apiClient.post<ApiEnvelope<unknown>>(`${base}/start`);
          return sleepSchema.parse(data.data);
        }
        case 'stop': {
          const { data } = await apiClient.post<ApiEnvelope<unknown>>(`${base}/${action.sleepId}/stop`, {});
          return sleepSchema.parse(data.data);
        }
        case 'manual': {
          const { data } = await apiClient.post<ApiEnvelope<unknown>>(base, {
            started_at: action.input.startedAt,
            ended_at: action.input.endedAt,
          });
          return sleepSchema.parse(data.data);
        }
        case 'delete':
          await apiClient.delete(`${base}/${action.sleepId}`);
          return null;
      }
    },
    onSuccess: (sleep) => {
      if (sleep) putSleep(queryClient, id, sleep);
    },
    onSettled: () => refresh(queryClient, id),
  });
}

export type DiaperAction = { kind: 'add'; diaper: DiaperKind } | { kind: 'delete'; diaperId: number };

export function useDiaperAction(id: number) {
  const queryClient = useQueryClient();
  return useMutation<BabyDiaper | null, unknown, DiaperAction>({
    mutationFn: async (action) => {
      const base = `/children/${id}/diapers`;
      if (action.kind === 'add') {
        const { data } = await apiClient.post<ApiEnvelope<unknown>>(base, { kind: action.diaper });
        return diaperSchema.parse(data.data);
      }
      await apiClient.delete(`${base}/${action.diaperId}`);
      return null;
    },
    onSettled: () => refresh(queryClient, id),
  });
}

/** 403 `child_read_only`: a spouse viewing a shared child — the screen turns read-only. */
export function isReadOnlyError(error: unknown): boolean {
  return getApiErrorCode(error) === 'child_read_only';
}

/** First validation message of `field` on a 422, if any. */
export function babyFieldError(error: unknown, field: string): string | undefined {
  if (!(error instanceof ApiError)) return undefined;
  const body = error.response?.data as ApiEnvelope<unknown> | undefined;
  const list = body && typeof body === 'object' ? body.errors?.[field] : undefined;
  return Array.isArray(list) && typeof list[0] === 'string' ? list[0] : undefined;
}

/**
 * What to show under a failed write: the first validation message, or the
 * server's localized message of a 4xx (409 `feed_active`, 403 `child_read_only`
 * …); undefined for network / 5xx errors (the caller's generic copy).
 */
export function babyActionError(error: unknown): string | undefined {
  if (!(error instanceof ApiError)) return undefined;
  const status = error.response?.status ?? 0;
  if (status < 400 || status >= 500) return undefined;
  const body = error.response?.data as ApiEnvelope<unknown> | undefined;
  const errors = body && typeof body === 'object' ? body.errors : undefined;
  if (errors && typeof errors === 'object') {
    for (const list of Object.values(errors)) {
      if (Array.isArray(list) && typeof list[0] === 'string') return list[0];
    }
  }
  return typeof body?.message === 'string' && body.message.trim() ? body.message : undefined;
}
