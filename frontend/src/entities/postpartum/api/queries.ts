'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback, useState } from 'react';

import { type ApiEnvelope, ApiError, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type {
  ActivatePostpartumInput,
  EpdsKind,
  EpdsQuestionnaire,
  EpdsResult,
  PostpartumOverview,
  PostpartumRecovery,
  RecoveryUpdate,
} from '../model/types';
import { postpartumKeys } from './keys';
import { epdsResultSchema, overviewSchema, questionnaireSchema, recoverySchema } from './schema';

/*
 * `/api/v1/postpartum*` (B-N5-01), Go only. Health data (CLAUDE.md §11):
 * never log a payload or a response; EPDS answers never enter a cache.
 */

export async function fetchPostpartum(): Promise<PostpartumOverview> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/postpartum');
  return overviewSchema.parse(data.data);
}

/** GET /postpartum — the home read model (status, today's recovery, alerts, tip, call-when). */
export function usePostpartum() {
  return useQuery({
    queryKey: postpartumKeys.overview(),
    queryFn: fetchPostpartum,
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

/** Snake-case body of the activation form. */
export function toActivateBody(input: ActivatePostpartumInput): Record<string, unknown> {
  return { birth_date: input.birthDate, delivery_type: input.deliveryType, baby_count: input.babyCount };
}

/**
 * POST /postpartum/activate — closes an active pregnancy as delivered and puts the user in postpartum mode.
 * The overview cache gets the answer; callers refresh the life-stage / pregnancy reads they own.
 */
export function useActivatePostpartum() {
  const queryClient = useQueryClient();
  return useMutation<PostpartumOverview, unknown, ActivatePostpartumInput>({
    mutationFn: async (input) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/postpartum/activate', toActivateBody(input));
      return overviewSchema.parse(data.data);
    },
    onSuccess: (overview) => {
      queryClient.setQueryData(postpartumKeys.overview(), overview);
      void queryClient.invalidateQueries({ queryKey: postpartumKeys.all });
    },
  });
}

export async function fetchRecovery(date: string | null): Promise<PostpartumRecovery> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/postpartum/recovery', {
    params: date ? { date } : undefined,
  });
  return recoverySchema.parse(data.data);
}

/** GET /postpartum/recovery?date — one day of the recovery log (today when `date` is null). */
export function usePostpartumRecovery(date: string | null = null) {
  return useQuery({
    queryKey: postpartumKeys.recovery(date),
    queryFn: () => fetchRecovery(date),
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

/** Snake-case partial body: only the keys present in `update` are sent (`null` clears). */
export function toRecoveryBody(update: RecoveryUpdate): Record<string, unknown> {
  const map: Record<keyof RecoveryUpdate, string> = {
    lochiaAmount: 'lochia_amount',
    lochiaColor: 'lochia_color',
    painLevel: 'pain_level',
    painLocations: 'pain_locations',
    breasts: 'breasts',
    feedsCount: 'feeds_count',
    sleepHours: 'sleep_hours',
  };
  const body: Record<string, unknown> = {};
  for (const key of Object.keys(map) as (keyof RecoveryUpdate)[]) {
    if (key in update) body[map[key]] = update[key];
  }
  return body;
}

/** PUT /postpartum/recovery — today's recovery (partial); the home refetches. */
export function useSaveRecovery() {
  const queryClient = useQueryClient();
  return useMutation<PostpartumRecovery, unknown, RecoveryUpdate>({
    mutationFn: async (update) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>('/postpartum/recovery', toRecoveryBody(update));
      return recoverySchema.parse(data.data);
    },
    onSuccess: (saved) => {
      queryClient.setQueryData(postpartumKeys.recovery(null), saved);
      void queryClient.invalidateQueries({ queryKey: postpartumKeys.overview() });
    },
  });
}

export async function fetchEpdsQuestions(kind: EpdsKind): Promise<EpdsQuestionnaire> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/postpartum/epds/questions', { params: { kind } });
  return questionnaireSchema.parse(data.data);
}

/** GET /postpartum/epds/questions?kind — the questionnaire in the request locale (text + option scores). */
export function useEpdsQuestions(kind: EpdsKind, locale: string) {
  return useQuery({
    queryKey: postpartumKeys.questions(kind, locale),
    queryFn: () => fetchEpdsQuestions(kind),
    enabled: isAuthenticated(),
    staleTime: 10 * 60_000,
    retry: 1,
  });
}

export async function postEpds(kind: EpdsKind, answers: Readonly<Record<string, number>>): Promise<EpdsResult> {
  const { data } = await apiClient.post<ApiEnvelope<unknown>>('/postpartum/epds', { kind, answers });
  return epdsResultSchema.parse(data.data);
}

export type EpdsSubmitState =
  | { status: 'idle' }
  | { status: 'pending' }
  | { status: 'error'; error: unknown }
  | { status: 'success'; result: EpdsResult };

/**
 * POST /postpartum/epds. Deliberately not a `useMutation`: a mutation keeps its
 * variables (the answers) in the mutation cache; here they live only in the
 * request. Only the scored result (totals + safety) is kept.
 */
export function useSubmitEpds() {
  const queryClient = useQueryClient();
  const [state, setState] = useState<EpdsSubmitState>({ status: 'idle' });
  const submit = useCallback(
    async (kind: EpdsKind, answers: Readonly<Record<string, number>>) => {
      setState({ status: 'pending' });
      try {
        const result = await postEpds(kind, answers);
        setState({ status: 'success', result });
        void queryClient.invalidateQueries({ queryKey: postpartumKeys.overview() });
        void queryClient.invalidateQueries({ queryKey: postpartumKeys.history() });
        return result;
      } catch (error) {
        setState({ status: 'error', error });
        return null;
      }
    },
    [queryClient],
  );
  const reset = useCallback(() => setState({ status: 'idle' }), []);
  return { state, submit, reset };
}

/** First validation message of `field` on a 422, if any. */
export function fieldError(error: unknown, field: string): string | undefined {
  if (!(error instanceof ApiError)) return undefined;
  const body = error.response?.data as ApiEnvelope<unknown> | undefined;
  const list = body && typeof body === 'object' ? body.errors?.[field] : undefined;
  return Array.isArray(list) && typeof list[0] === 'string' ? list[0] : undefined;
}
