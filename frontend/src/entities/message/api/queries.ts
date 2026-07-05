'use client';

import { useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { DailyMessage, UserMode } from '../model/types';
import { dailyMessageSchema, userModeSchema } from './schema';

/**
 * Query-key factory for personalized messages (CLAUDE.md §8). The daily message
 * key includes the date so each day is cached separately.
 */
export const messageKeys = {
  all: ['message'] as const,
  mode: () => [...messageKeys.all, 'mode'] as const,
  daily: (date?: string) => [...messageKeys.all, 'daily', date ?? 'today'] as const,
};

/** GET /messages/mode — the user's current mode and subscription flags. */
export async function fetchUserMode(): Promise<UserMode> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/messages/mode');
  return userModeSchema.parse(data.data);
}

/** Reads the current mode (§8 — server state stays in TanStack Query). */
export function useUserMode() {
  return useQuery({
    queryKey: messageKeys.mode(),
    queryFn: fetchUserMode,
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: false,
  });
}

/**
 * GET /messages/daily — the day's personalized message bundle.
 *
 * Privacy (§11): the date is the only parameter and carries no health data; the
 * response text is display-only and must never be logged.
 */
export async function fetchDailyMessage(date?: string): Promise<DailyMessage> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/messages/daily', {
    params: date ? { date } : undefined,
  });
  return dailyMessageSchema.parse(data.data);
}

/** Reads today's (or a given day's) personalized message. */
export function useDailyMessage(date?: string) {
  return useQuery({
    queryKey: messageKeys.daily(date),
    queryFn: () => fetchDailyMessage(date),
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: false,
  });
}
