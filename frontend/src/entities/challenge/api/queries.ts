'use client';

import { useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { TodayChallenge } from '../model/types';
import { challengeSectionSchema } from './schema';

/** Query-key factory for the daily challenge (CLAUDE.md §8). */
export const challengeKeys = {
  all: ['challenge'] as const,
  today: (date?: string) => [...challengeKeys.all, 'today', date ?? 'today'] as const,
};

/**
 * GET /home/sections/challenge — today's challenge for this user.
 *
 * Returns `null` when no challenge is available (empty or fully inactive pool),
 * which the card treats as "render nothing".
 */
export async function fetchTodayChallenge(date?: string): Promise<TodayChallenge | null> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/home/sections/challenge', {
    params: date ? { date } : undefined,
  });
  return challengeSectionSchema.parse(data.data);
}

/**
 * Today's challenge. The pick is stable for the whole day server-side, so this
 * stays fresh for a few minutes; the toggle mutation writes the authoritative
 * result straight into this cache entry.
 */
export function useTodayChallenge(date?: string) {
  return useQuery({
    queryKey: challengeKeys.today(date),
    queryFn: () => fetchTodayChallenge(date),
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: false,
  });
}
