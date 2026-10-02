'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { ApiError, type ApiEnvelope, apiClient, getApiErrorStatus } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type {
  LossCatalogGroup,
  LossCatalogItem,
  LossFollowupInput,
  LossMood,
  LossNextStep,
  LossNote,
  LossState,
  RecordLossInput,
} from '../model/types';
import { lossKeys } from './keys';
import { lossCatalogSchema, lossNoteSchema, lossStateSchema } from './schema';

/*
 * `/api/v1/loss/*` (CB-LOSS-01), Go only. Health data of the most sensitive
 * kind (§11): never log a payload, a response or an error body, and never put
 * any of it in a URL. Every write answers the full state, which replaces the
 * cached one.
 */

export async function fetchLossState(): Promise<LossState> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/loss');
  return lossStateSchema.parse(data.data);
}

/** GET /loss — the latest loss with its follow-up, moods and note flag (`loss` null when none). */
export function useLossState() {
  return useQuery({
    queryKey: lossKeys.state(),
    queryFn: fetchLossState,
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

function useStateWrite<V>(send: (vars: V) => Promise<ApiEnvelope<unknown>>) {
  const queryClient = useQueryClient();
  return useMutation<LossState, unknown, V>({
    mutationFn: async (vars) => lossStateSchema.parse((await send(vars)).data),
    onSuccess: (state) => queryClient.setQueryData(lossKeys.state(), state),
  });
}

/** POST /loss — records (or, the same day, corrects) the loss; pregnancy content stops server-side. */
export function useRecordLoss() {
  return useStateWrite<RecordLossInput>(async (input) => {
    const body: Record<string, unknown> = {};
    if (input.type !== undefined) body.type = input.type;
    if (input.occurredOn !== undefined) body.occurred_on = input.occurredOn;
    if (input.notifyCompanion !== undefined) body.notify_companion = input.notifyCompanion;
    return (await apiClient.post<ApiEnvelope<unknown>>('/loss', body)).data;
  });
}

/** PUT /loss/followup — only the given keys. */
export function useSaveLossFollowup() {
  return useStateWrite<LossFollowupInput>(async (input) => {
    const body: Record<string, unknown> = {};
    if (input.bleedingStopped !== undefined) body.bleeding_stopped = input.bleedingStopped;
    if (input.betaNextOn !== undefined) body.beta_next_on = input.betaNextOn;
    if (input.betaNegative !== undefined) body.beta_negative = input.betaNegative;
    if (input.visitAt !== undefined) body.visit_at = input.visitAt;
    return (await apiClient.put<ApiEnvelope<unknown>>('/loss/followup', body)).data;
  });
}

/** POST /loss/moods — today's «امروز چطوری؟» (one per day; a new tap replaces it). */
export function useLogLossMood() {
  return useStateWrite<LossMood>(
    async (mood) => (await apiClient.post<ApiEnvelope<unknown>>('/loss/moods', { mood })).data,
  );
}

/** PUT /loss/next-step — also switches the life mode server-side (cycle / ttc). */
export function useSaveLossNextStep() {
  return useStateWrite<LossNextStep>(
    async (choice) => (await apiClient.put<ApiEnvelope<unknown>>('/loss/next-step', { choice })).data,
  );
}

/** The first field message of a 422 (localized by the API), e.g. «این تاریخ نمی‌تواند در آینده باشد.». */
export function lossFieldError(error: unknown): string | undefined {
  if (!(error instanceof ApiError) || error.response?.status !== 422) return undefined;
  const body = error.response.data as ApiEnvelope<unknown> | undefined;
  const errors = body && typeof body === 'object' ? body.errors : undefined;
  if (!errors || typeof errors !== 'object') return undefined;
  for (const list of Object.values(errors)) {
    const first = Array.isArray(list) ? list.find((m) => typeof m === 'string' && m.trim()) : undefined;
    if (first) return first;
  }
  return undefined;
}

/**
 * Why the private note can't be used: `unavailable` = 503 `note_unavailable`
 * (no server key — notes are switched off), `unreadable` = 409 (she may delete it).
 */
export type LossNoteProblem = 'unavailable' | 'unreadable' | 'missing' | 'other';

export function lossNoteProblem(error: unknown): LossNoteProblem {
  const status = getApiErrorStatus(error);
  if (status === 503) return 'unavailable';
  if (status === 409) return 'unreadable';
  if (status === 404) return 'missing';
  return 'other';
}

/** GET /loss/note — decrypted only for her; fetched only while the note sheet is open, never cached after. */
export function useLossNote(enabled: boolean) {
  return useQuery<LossNote>({
    queryKey: lossKeys.note(),
    queryFn: async () => lossNoteSchema.parse((await apiClient.get<ApiEnvelope<unknown>>('/loss/note')).data.data),
    enabled: enabled && isAuthenticated(),
    gcTime: 0,
    staleTime: 0,
    retry: false,
  });
}

/** PUT /loss/note — blank clears it. DELETE /loss/note when `note` is null (also removes an unreadable one). */
export function useSaveLossNote() {
  const queryClient = useQueryClient();
  return useMutation<LossNote, unknown, string | null>({
    mutationFn: async (note) => {
      const { data } = note
        ? await apiClient.put<ApiEnvelope<unknown>>('/loss/note', { note })
        : await apiClient.delete<ApiEnvelope<unknown>>('/loss/note');
      return lossNoteSchema.catch({ note: null, updatedAt: null }).parse(data.data ?? {});
    },
    onSuccess: (note) => {
      queryClient.setQueryData(lossKeys.note(), note);
      void queryClient.invalidateQueries({ queryKey: lossKeys.state() });
    },
  });
}

/** GET /catalog/<loss group> — admin-editable copy in the request locale (content, not health data). */
export function useLossCatalog(group: LossCatalogGroup, locale: string) {
  return useQuery<LossCatalogItem[]>({
    queryKey: lossKeys.catalog(group, locale),
    queryFn: async () => lossCatalogSchema.parse((await apiClient.get<ApiEnvelope<unknown>>(`/catalog/${group}`)).data.data),
    enabled: isAuthenticated(),
    staleTime: 10 * 60_000,
    retry: 1,
  });
}
