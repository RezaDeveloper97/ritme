'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type {
  HotFlashDetails,
  MenopauseFlash,
  MenopauseFlashDay,
  MenopauseMessage,
  MenopauseProfile,
  MenopauseProfileUpdate,
  MenopauseToday,
} from '../model/types';
import { toHotFlashDetailsBody } from './hot-flashes';
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
 * The hot-flash timer (home tile + `/menopause/hot-flash`): `start` = POST
 * /menopause/hot-flashes without a duration (the running one comes back if
 * there is one; details may ride along), `stop` = POST
 * /menopause/hot-flashes/{id}/stop — on a stopped flash the same call only
 * edits its details (CB-MENO-07). Both refresh the home and the day list; the
 * day cache gets the flash at once so the ring never waits for the refetch.
 */
export function useHotFlashTimer() {
  const queryClient = useQueryClient();
  const refresh = (flash: MenopauseFlash) => {
    queryClient.setQueryData<MenopauseFlashDay>(menopauseKeys.hotFlashDay(null), (day) =>
      day ? { ...day, running: flash.running ? flash : null } : day,
    );
    void queryClient.invalidateQueries({ queryKey: menopauseKeys.today() });
    void queryClient.invalidateQueries({ queryKey: menopauseKeys.hotFlashes() });
  };
  const start = useMutation<MenopauseFlash, unknown, HotFlashDetails | void>({
    mutationFn: async (details) => {
      const body = details ? toHotFlashDetailsBody(details) : {};
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/menopause/hot-flashes', body);
      return menopauseFlashSchema.parse(data.data);
    },
    onSuccess: refresh,
  });
  const stop = useMutation<MenopauseFlash, unknown, number | { id: number; details: HotFlashDetails }>({
    mutationFn: async (target) => {
      const id = typeof target === 'number' ? target : target.id;
      const body = typeof target === 'number' ? {} : toHotFlashDetailsBody(target.details);
      const { data } = await apiClient.post<ApiEnvelope<unknown>>(`/menopause/hot-flashes/${id}/stop`, body);
      return menopauseFlashSchema.parse(data.data);
    },
    onSuccess: refresh,
  });
  return { start, stop };
}
