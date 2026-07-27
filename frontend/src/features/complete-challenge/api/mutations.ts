'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import {
  challengeKeys,
  challengeToggleSchema,
  type ChallengeToggleResult,
  type TodayChallenge,
} from '@/entities/challenge';
import { type ApiEnvelope, apiClient } from '@/shared/api';

/**
 * Tick today's challenge off (or undo it).
 *
 * The checkbox must feel instant, so the cache flips optimistically and is then
 * reconciled with the server's authoritative streak. The previous value is kept
 * so a failed request rolls the card back rather than leaving a false tick.
 */
export function useToggleChallenge(date?: string) {
  const queryClient = useQueryClient();
  const key = challengeKeys.today(date);

  return useMutation<
    ChallengeToggleResult,
    unknown,
    number,
    { previous: TodayChallenge | null | undefined }
  >({
    mutationFn: async (challengeId) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>(
        `/home/challenges/${challengeId}/toggle`,
      );
      return challengeToggleSchema.parse(data.data);
    },

    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: key });
      const previous = queryClient.getQueryData<TodayChallenge | null>(key);

      queryClient.setQueryData<TodayChallenge | null>(key, (current) =>
        current ? optimistic(current) : current,
      );

      return { previous };
    },

    onError: (_error, _id, context) => {
      if (context) queryClient.setQueryData(key, context.previous);
    },

    onSuccess: (result) => {
      queryClient.setQueryData<TodayChallenge | null>(key, (current) =>
        current
          ? {
              ...current,
              isCompleted: result.isCompleted,
              streak: result.streak,
              longestStreak: result.longestStreak,
              weekDays: result.weekDays,
              statusMessage: result.statusMessage ?? current.description,
            }
          : current,
      );
    },
  });
}

/**
 * Local guess at the post-toggle state: flip the tick, move today's dot, and
 * step the streak by one in the matching direction (never below zero).
 */
function optimistic(current: TodayChallenge): TodayChallenge {
  const isCompleted = !current.isCompleted;

  return {
    ...current,
    isCompleted,
    streak: Math.max(0, current.streak + (isCompleted ? 1 : -1)),
    weekDays: current.weekDays.map((day) =>
      day.isToday ? { ...day, isCompleted } : day,
    ),
  };
}
