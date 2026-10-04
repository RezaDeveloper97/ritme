'use client';

import { type QueryClient, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { pregnancyKeys } from '@/entities/pregnancy';
import { type ApiEnvelope, apiClient, getApiErrorStatus } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { ContractionOverview, ContractionSession, KickOverview, KickSession } from '../model/types';
import {
  parseContractionOverview,
  parseContractionSession,
  parseKickOverview,
  parseKickSession,
  parseStopResult,
} from './schema';

/*
 * `/api/v1/pregnancy/kick-sessions` and `/api/v1/pregnancy/contractions`
 * (B-N5-03). The server holds the running session, so a reload restores it from
 * these reads; every mutation writes its returned session straight into the
 * overview cache. Privacy (§11): never log these payloads.
 */

/** Query keys, nested under the pregnancy entity so a mode switch refreshes them. */
export const pregnancyToolKeys = {
  all: () => [...pregnancyKeys.all, 'tools'] as const,
  kicks: () => [...pregnancyKeys.all, 'tools', 'kicks'] as const,
  contractions: () => [...pregnancyKeys.all, 'tools', 'contractions'] as const,
};

const KICKS = '/pregnancy/kick-sessions';
const CONTRACTIONS = '/pregnancy/contractions';

async function get<T>(path: string, parse: (raw: unknown, receivedAt: number) => T): Promise<T> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(path);
  return parse(data.data, Date.now());
}

async function post<T>(path: string, parse: (raw: unknown, receivedAt: number) => T): Promise<T> {
  const { data } = await apiClient.post<ApiEnvelope<unknown>>(path);
  return parse(data.data, Date.now());
}

async function del<T>(path: string, parse: (raw: unknown, receivedAt: number) => T): Promise<T> {
  const { data } = await apiClient.delete<ApiEnvelope<unknown>>(path);
  return parse(data.data, Date.now());
}

/** 409 = the server state moved on (another tab started/stopped, or pregnancy mode is off). */
export function isConflict(error: unknown): boolean {
  return getApiErrorStatus(error) === 409;
}

// ── kick counter ───────────────────────────────────────────────

export function fetchKickOverview(): Promise<KickOverview> {
  return get(KICKS, parseKickOverview);
}

export function useKickOverview() {
  return useQuery({
    queryKey: pregnancyToolKeys.kicks(),
    queryFn: fetchKickOverview,
    enabled: isAuthenticated(),
    staleTime: 30_000,
    // A session left open in another tab shows up when the user comes back.
    refetchOnWindowFocus: true,
    retry: 1,
  });
}

export const kickApi = {
  start: () => post(KICKS, parseKickSession),
  kick: (id: number) => post(`${KICKS}/${id}/kicks`, parseKickSession),
  undo: (id: number) => del(`${KICKS}/${id}/kicks`, parseKickSession),
  stop: (id: number) => post(`${KICKS}/${id}/stop`, parseKickSession),
};

/** Writes a session the server just returned into the overview (`active` or history). */
export function putKickSession(queryClient: QueryClient, s: KickSession): void {
  queryClient.setQueryData<KickOverview>(pregnancyToolKeys.kicks(), (o) => {
    if (!o) return o;
    if (s.isActive) return { ...o, active: s };
    return {
      ...o,
      active: o.active?.id === s.id ? null : o.active,
      history: [s, ...o.history.filter((h) => h.id !== s.id)],
    };
  });
}

export function useStopKicks() {
  const queryClient = useQueryClient();
  return useMutation<KickSession, unknown, number>({
    mutationFn: (id) => kickApi.stop(id),
    onSuccess: (s) => {
      putKickSession(queryClient, s);
      // The stop writes the day total into the fetal-movement log (v1 tab, v2 day, analysis).
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.all });
    },
    onError: (error) => {
      if (isConflict(error)) void queryClient.invalidateQueries({ queryKey: pregnancyToolKeys.kicks() });
    },
  });
}

// ── contraction timer ──────────────────────────────────────────

export function fetchContractionOverview(): Promise<ContractionOverview> {
  return get(CONTRACTIONS, parseContractionOverview);
}

export function useContractionOverview() {
  return useQuery({
    queryKey: pregnancyToolKeys.contractions(),
    queryFn: fetchContractionOverview,
    enabled: isAuthenticated(),
    staleTime: 30_000,
    refetchOnWindowFocus: true,
    retry: 1,
  });
}

function putContractionSession(queryClient: QueryClient, s: ContractionSession): void {
  queryClient.setQueryData<ContractionOverview>(pregnancyToolKeys.contractions(), (o) => {
    if (!o) return o;
    if (s.isActive) return { ...o, active: s };
    return {
      ...o,
      active: o.active?.id === s.id ? null : o.active,
      history: [s, ...o.history.filter((h) => h.id !== s.id)],
    };
  });
}

/** Start (`running` = false) or stop (`running` = true) the contraction being timed. */
export function useToggleContraction() {
  const queryClient = useQueryClient();
  return useMutation<
    { session: ContractionSession; alerts: ReturnType<typeof parseStopResult>['alerts'] },
    unknown,
    { running: boolean }
  >({
    mutationFn: async ({ running }) =>
      running
        ? post(`${CONTRACTIONS}/stop`, parseStopResult)
        : { session: await post(`${CONTRACTIONS}/start`, parseContractionSession), alerts: [] },
    onSuccess: ({ session, alerts }) => {
      putContractionSession(queryClient, session);
      // A raised 5-1-1 alert also lands in the alert centre and the Today badge.
      if (alerts.length) void queryClient.invalidateQueries({ queryKey: pregnancyKeys.v2.all() });
    },
    onError: (error) => {
      if (isConflict(error)) void queryClient.invalidateQueries({ queryKey: pregnancyToolKeys.contractions() });
    },
  });
}

/** «توقف و ذخیره»: closes the session (a contraction still running is ended by the server). */
export function useFinishContractions() {
  const queryClient = useQueryClient();
  return useMutation<ContractionSession, unknown, number>({
    mutationFn: (id) => post(`${CONTRACTIONS}/sessions/${id}/finish`, parseContractionSession),
    onSuccess: (s) => putContractionSession(queryClient, s),
    onError: (error) => {
      if (isConflict(error)) void queryClient.invalidateQueries({ queryKey: pregnancyToolKeys.contractions() });
    },
  });
}

/** «دیدم، ممنون» on the 5-1-1 card: POST /pregnancy/v2/alerts/{id}/actions/ack. */
export function useAckToolAlert() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, number>({
    mutationFn: async (id) => {
      await apiClient.post(`/pregnancy/v2/alerts/${id}/actions/ack`);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.v2.all() });
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.alertSummary() });
    },
  });
}
