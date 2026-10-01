'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { LIFE_MODES, type LifeStage, type LifeStageUpdate, type LossCopy } from '../model/life-stage';
import { userKeys } from './queries';

/** Query keys of the life-stage reads (under the account's `user` root). */
export const lifeStageKeys = {
  all: [...userKeys.all, 'life-stage'] as const,
  stage: () => [...lifeStageKeys.all, 'stage'] as const,
  lossCopy: () => [...lifeStageKeys.all, 'loss-copy'] as const,
};

export const lifeStageSchema = z
  .object({
    mode: z.enum(LIFE_MODES),
    stored_mode: z.enum(LIFE_MODES).nullable().catch(null),
    ivf_iui: z.boolean().catch(false),
    track_contraception: z.boolean().catch(false),
  })
  .transform(
    (d): LifeStage => ({
      mode: d.mode,
      storedMode: d.stored_mode,
      ivfIui: d.ivf_iui,
      trackContraception: d.track_contraception,
    }),
  );

const text = z.string().nullable().catch(null);

export const lossCopySchema = z
  .object({
    title: text,
    body: text,
    confirm: text,
    cancel: text,
    done_title: text,
    done_body: text,
    done_action: text,
  })
  .transform(
    (d): LossCopy => ({
      title: d.title,
      body: d.body,
      confirm: d.confirm,
      cancel: d.cancel,
      doneTitle: d.done_title,
      doneBody: d.done_body,
      doneAction: d.done_action,
    }),
  );

/** GET /profile/life-stage — effective + stored mode and the two switches (Go only). */
export async function fetchLifeStage(): Promise<LifeStage> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/profile/life-stage');
  return lifeStageSchema.parse(data.data);
}

/**
 * The user's life stage. Go-only endpoint: where it is missing (production still
 * on Laravel) the query errors and callers fall back to `/messages/mode`.
 */
export function useLifeStage(options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: lifeStageKeys.stage(),
    queryFn: fetchLifeStage,
    enabled: isAuthenticated() && (options.enabled ?? true),
    staleTime: 5 * 60_000,
    retry: false,
  });
}

/** Snake-case body of a partial update; only the keys that were given. */
export function toLifeStageBody(update: LifeStageUpdate): Record<string, unknown> {
  const body: Record<string, unknown> = {};
  if (update.mode !== undefined) body.mode = update.mode;
  if (update.ivfIui !== undefined) body.ivf_iui = update.ivfIui;
  if (update.trackContraception !== undefined) body.track_contraception = update.trackContraception;
  return body;
}

/**
 * PUT /profile/life-stage — partial update. Optimistic for the two switches; the
 * saved state replaces the cache. A mode change reshapes the whole app (nav,
 * home, messages, TTC layout), so the caller invalidates broadly after it.
 */
export function useUpdateLifeStage() {
  const queryClient = useQueryClient();
  return useMutation<LifeStage, unknown, LifeStageUpdate, { previous: LifeStage | undefined }>({
    mutationFn: async (update) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>('/profile/life-stage', toLifeStageBody(update));
      return lifeStageSchema.parse(data.data);
    },
    onMutate: async (update) => {
      const key = lifeStageKeys.stage();
      await queryClient.cancelQueries({ queryKey: key });
      const previous = queryClient.getQueryData<LifeStage>(key);
      if (previous && update.mode === undefined) {
        queryClient.setQueryData<LifeStage>(key, {
          ...previous,
          ivfIui: update.ivfIui ?? previous.ivfIui,
          trackContraception: update.trackContraception ?? previous.trackContraception,
        });
      }
      return { previous };
    },
    onError: (_error, _update, context) => {
      if (context?.previous) queryClient.setQueryData(lifeStageKeys.stage(), context.previous);
    },
    onSuccess: (saved) => {
      queryClient.setQueryData(lifeStageKeys.stage(), saved);
    },
  });
}

/** GET /profile/life-stage/loss-copy — admin-edited calm-exit copy (texts may be null). */
export async function fetchLossCopy(): Promise<LossCopy> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/profile/life-stage/loss-copy');
  return lossCopySchema.parse(data.data);
}

export function useLossCopy() {
  return useQuery({
    queryKey: lifeStageKeys.lossCopy(),
    queryFn: fetchLossCopy,
    enabled: isAuthenticated(),
    staleTime: 10 * 60_000,
    retry: false,
  });
}
