'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import {
  type ChronicCondition,
  type PregnancyIntention,
  type UserProfile,
  userKeys,
  userProfileSchema,
} from '@/entities/user';

/**
 * Partial update payload for `POST /profile` — the API accepts any subset of
 * these fields and rejects unknown keys (422). Field names are the API's
 * snake_case; dates cross the boundary as Gregorian `YYYY-MM-DD` (CLAUDE.md §7)
 * and are produced only via `@/shared/lib/date`.
 *
 * This is sensitive health data (§11): it is sent to the API and never logged.
 */
export interface UpdateProfileInput {
  name?: string;
  /** Date of birth, Gregorian `YYYY-MM-DD`, must be before today. */
  birthday?: string;
  /** Weight in kilograms (20–300). */
  weight?: number;
  /** Height in centimetres (integer, 50–250). */
  height?: number;
  /** Typical period length in days (integer, 1–15). */
  period_duration?: number;
  /** Typical cycle length in days (integer, 15–60). */
  cycle_duration?: number;
  /** Start of the last period, Gregorian `YYYY-MM-DD`, not in the future. */
  last_period_start?: string;
  /** Pregnancy intention stated at onboarding; the API derives `user_goal`. */
  pregnancy_intention?: PregnancyIntention;
  /** Optional self-reported chronic conditions. */
  chronic_conditions?: ChronicCondition[];
}

/**
 * POST /profile — saves a partial profile update. The response echoes the full
 * `{ user, profile }` payload, validated at the boundary with the entity's zod
 * schema (§10). On success the profile and current-user caches are invalidated
 * via the entity's key factory (§8) so every consumer refetches fresh data.
 */
export function useUpdateProfile() {
  const queryClient = useQueryClient();
  return useMutation<UserProfile, unknown, UpdateProfileInput>({
    mutationFn: async (input) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/profile', input);
      return userProfileSchema.parse(data.data);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: userKeys.profile() });
      void queryClient.invalidateQueries({ queryKey: userKeys.current() });
    },
  });
}
