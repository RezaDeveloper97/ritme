'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type {
  MenopauseFlash,
  MenopauseMessage,
  MenopauseProfile,
  MenopauseProfileUpdate,
  MenopauseToday,
} from '../model/types';
import { menopauseKeys } from './keys';
import {
  menopauseFlashSchema,
  menopauseMessagesSchema,
  menopauseProfileSchema,
  menopauseTodaySchema,
} from './schema';

/*
 * `/api/v1/menopause/*` (CB-MENO-02) and `/messages/menopause` (CB-MENO-12),
 * Go only. Health data (§11): never log a payload or a response.
 */

export async function fetchMenopauseProfile(): Promise<MenopauseProfile> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/menopause/profile');
  return menopauseProfileSchema.parse(data.data);
}

/** GET /menopause/profile — the stage answers with the stage rule applied. */
export function useMenopauseProfile() {
  return useQuery({
    queryKey: menopauseKeys.profile(),
    queryFn: fetchMenopauseProfile,
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: 1,
  });
}

export async function fetchMenopauseToday(): Promise<MenopauseToday> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/menopause/today');
  return menopauseTodaySchema.parse(data.data);
}

/** GET /menopause/today — the home read model (Meno_Home / Main). */
export function useMenopauseToday() {
  return useQuery({
    queryKey: menopauseKeys.today(),
    queryFn: fetchMenopauseToday,
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

export async function fetchMenopauseMessages(): Promise<MenopauseMessage[]> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/messages/menopause');
  return menopauseMessagesSchema.parse(data.data);
}

/** GET /messages/menopause — alerts, reminders and stage tips of the home, in display order. */
export function useMenopauseMessages() {
  return useQuery({
    queryKey: menopauseKeys.messages(),
    queryFn: fetchMenopauseMessages,
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: false,
  });
}

/** Snake-case body of the stage screen. */
export function toMenopauseProfileBody(update: MenopauseProfileUpdate): Record<string, unknown> {
  return {
    stage: update.stage,
    last_period: update.lastPeriod,
    surgical: update.surgical,
    hrt: update.hrt,
  };
}

/** PUT /menopause/profile — the saved profile replaces the cache; today and messages refetch. */
export function useSaveMenopauseProfile() {
  const queryClient = useQueryClient();
  return useMutation<MenopauseProfile, unknown, MenopauseProfileUpdate>({
    mutationFn: async (update) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>('/menopause/profile', toMenopauseProfileBody(update));
      return menopauseProfileSchema.parse(data.data);
    },
    onSuccess: (saved) => {
      queryClient.setQueryData(menopauseKeys.profile(), saved);
      void queryClient.invalidateQueries({ queryKey: menopauseKeys.today() });
      void queryClient.invalidateQueries({ queryKey: menopauseKeys.messages() });
    },
  });
}

/**
 * The hot-flash timer from the home: `start` = POST /menopause/hot-flashes
 * without a duration (the running one comes back if there is one), `stop` =
 * POST /menopause/hot-flashes/{id}/stop. Both refresh the home.
 */
export function useHotFlashTimer() {
  const queryClient = useQueryClient();
  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: menopauseKeys.today() });
  };
  const start = useMutation<MenopauseFlash, unknown, void>({
    mutationFn: async () => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/menopause/hot-flashes', {});
      return menopauseFlashSchema.parse(data.data);
    },
    onSuccess: refresh,
  });
  const stop = useMutation<MenopauseFlash, unknown, number>({
    mutationFn: async (id) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>(`/menopause/hot-flashes/${id}/stop`, {});
      return menopauseFlashSchema.parse(data.data);
    },
    onSuccess: refresh,
  });
  return { start, stop };
}
