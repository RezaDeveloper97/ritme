'use client';

import { useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { WeeklyWellbeing } from '../model/types';
import { weeklyWellbeingSectionSchema } from './schema';

/** Query-key factory for weekly wellbeing (§8). */
export const wellbeingKeys = {
  all: ['wellbeing'] as const,
  weekly: (date?: string) => [...wellbeingKeys.all, 'weekly', date ?? 'today'] as const,
};

/**
 * GET /home/sections/weekly_summary — mood/sleep/energy averaged over the seven
 * days ending on `date` (today when omitted). Returns `null` when the section
 * has nothing to show.
 */
export async function fetchWeeklyWellbeing(date?: string): Promise<WeeklyWellbeing | null> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/home/sections/weekly_summary', {
    params: date ? { date } : undefined,
  });
  return weeklyWellbeingSectionSchema.parse(data.data);
}

/**
 * The week's wellbeing summary. It only moves when the user saves a log, so it
 * stays fresh for a few minutes; saving a log invalidates this key so the card
 * catches up straight away.
 */
export function useWeeklyWellbeing(date?: string) {
  return useQuery({
    queryKey: wellbeingKeys.weekly(date),
    queryFn: () => fetchWeeklyWellbeing(date),
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: false,
  });
}
