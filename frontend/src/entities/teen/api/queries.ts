'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { TeenKit, TeenProfileInput, TeenProfileState, TeenToday } from '../model/types';
import { teenKeys } from './keys';
import { teenKitSchema, teenProfileStateSchema, teenTodaySchema } from './schema';

/*
 * `/api/v1/teen/*` (CB-TEEN-01), Go only. A minor's health data
 * (CLAUDE.md §11): never log a payload or a response.
 */

export async function fetchTeenProfile(): Promise<TeenProfileState> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/teen/profile');
  return teenProfileStateSchema.parse(data.data);
}

/** GET /teen/profile — the onboarding answers (null before onboarding). */
export function useTeenProfile() {
  return useQuery({
    queryKey: teenKeys.profile(),
    queryFn: fetchTeenProfile,
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: 1,
  });
}

export async function fetchTeenToday(): Promise<TeenToday> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/teen/today');
  return teenTodaySchema.parse(data.data);
}

/** GET /teen/today — the teen home read model. */
export function useTeenToday() {
  return useQuery({
    queryKey: teenKeys.today(),
    queryFn: fetchTeenToday,
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

/**
 * PUT /teen/profile — the onboarding answers. The caller switches the life
 * stage to teen first (`PUT /profile/life-stage`, entities/user).
 */
export function useSaveTeenProfile() {
  const queryClient = useQueryClient();
  return useMutation<TeenProfileState, unknown, TeenProfileInput>({
    mutationFn: async (input) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>('/teen/profile', {
        age_band: input.ageBand,
        menarche: input.menarche,
      });
      return teenProfileStateSchema.parse(data.data);
    },
    onSuccess: (saved) => {
      queryClient.setQueryData(teenKeys.profile(), saved);
      void queryClient.invalidateQueries({ queryKey: teenKeys.today() });
    },
  });
}

interface KitToggle {
  code: string;
  checked: boolean;
}

/** Apply one tick to a cached checklist (optimistic UI). */
export function withKitTick(kit: TeenKit, { code, checked }: KitToggle): TeenKit {
  const items = kit.items.map((item) => (item.code === code ? { ...item, checked } : item));
  const checkedCount = items.filter((item) => item.checked).length;
  return { ...kit, items, checkedCount, ready: items.length > 0 && checkedCount === items.length };
}

/** PUT /teen/kit/{code} — tick / untick a school-kit item; the home's checklist updates at once. */
export function useToggleTeenKit() {
  const queryClient = useQueryClient();
  return useMutation<TeenKit, unknown, KitToggle, { previous: TeenToday | undefined }>({
    mutationFn: async ({ code, checked }) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(`/teen/kit/${encodeURIComponent(code)}`, { checked });
      return teenKitSchema.parse(data.data);
    },
    onMutate: async (toggle) => {
      const key = teenKeys.today();
      await queryClient.cancelQueries({ queryKey: key });
      const previous = queryClient.getQueryData<TeenToday>(key);
      if (previous) queryClient.setQueryData<TeenToday>(key, { ...previous, kit: withKitTick(previous.kit, toggle) });
      return { previous };
    },
    onError: (_error, _toggle, context) => {
      if (context?.previous) queryClient.setQueryData(teenKeys.today(), context.previous);
    },
    onSuccess: (kit) => {
      const current = queryClient.getQueryData<TeenToday>(teenKeys.today());
      if (current) queryClient.setQueryData<TeenToday>(teenKeys.today(), { ...current, kit });
    },
  });
}
