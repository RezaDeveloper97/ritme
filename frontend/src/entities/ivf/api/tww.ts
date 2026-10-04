'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { IvfOutcome, IvfOutcomeResult, IvfTww, IvfTwwMood } from '../model/tww';
import { ivfKeys } from './keys';
import { ivfDangerSignsSchema, ivfOutcomeSchema, ivfTwwSchema } from './tww-schema';

/*
 * The two-week wait + the cycle outcome (CB-IVF-05), Go only. Health data
 * (CLAUDE.md §11): never log a payload, a mood or a result.
 */

/** Query keys of CB-IVF-05 (kept here, beside CB-IVF-04's own, so `keys.ts` stays untouched). */
export const ivfTwwKeys = {
  tww: () => [...ivfKeys.all, 'tww'] as const,
  dangerSigns: (locale: string) => ivfKeys.catalog('ivf_danger_signs', locale),
};

export async function fetchIvfTww(): Promise<IvfTww> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/ivf/tww');
  return ivfTwwSchema.parse(data.data);
}

/** GET /ivf/tww — days since transfer / to the beta test, today's mood, luteal-support doses. */
export function useIvfTww() {
  return useQuery({
    queryKey: ivfTwwKeys.tww(),
    queryFn: fetchIvfTww,
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

/** Catalog `ivf_danger_signs` (OHSS, fever after a procedure; hotline 115) in the request locale. */
export function useIvfDangerSigns(locale: string) {
  return useQuery({
    queryKey: ivfTwwKeys.dangerSigns(locale),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/catalog/ivf_danger_signs', {
        params: { audience: 'ttc' },
      });
      return ivfDangerSignsSchema.parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 10 * 60_000,
    retry: 1,
  });
}

/**
 * PUT /ivf/tww/{date} {mood} — today's check-in chip (`null` clears it).
 * Optimistic: the chip moves at once and rolls back on an error.
 */
export function useSetIvfTwwMood() {
  const queryClient = useQueryClient();
  return useMutation<IvfTww, unknown, { date: string; mood: IvfTwwMood | null }, { previous?: IvfTww }>({
    mutationFn: async ({ date, mood }) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(`/ivf/tww/${date}`, { mood });
      return ivfTwwSchema.parse(data.data);
    },
    onMutate: async ({ date, mood }) => {
      await queryClient.cancelQueries({ queryKey: ivfTwwKeys.tww() });
      const previous = queryClient.getQueryData<IvfTww>(ivfTwwKeys.tww());
      if (previous && previous.today === date) {
        queryClient.setQueryData<IvfTww>(ivfTwwKeys.tww(), { ...previous, todayMood: mood });
      }
      return { previous };
    },
    onError: (_error, _input, context) => {
      if (context?.previous) queryClient.setQueryData(ivfTwwKeys.tww(), context.previous);
    },
    onSuccess: (tww) => queryClient.setQueryData(ivfTwwKeys.tww(), tww),
  });
}

/**
 * POST /ivf/cycles/current/outcome {result} — closes the open cycle (the API
 * stops the cycle's reminders unless positive). The caller shows the answer;
 * the IVF reads are refreshed behind it.
 */
export function useRecordIvfOutcome() {
  const queryClient = useQueryClient();
  return useMutation<IvfOutcome, unknown, IvfOutcomeResult>({
    mutationFn: async (result) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/ivf/cycles/current/outcome', { result });
      return ivfOutcomeSchema.parse(data.data);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ivfKeys.home() });
      void queryClient.invalidateQueries({ queryKey: ivfKeys.meds() });
      void queryClient.invalidateQueries({ queryKey: ivfTwwKeys.tww() });
    },
  });
}
