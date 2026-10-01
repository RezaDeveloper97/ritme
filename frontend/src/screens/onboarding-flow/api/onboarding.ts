'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { userKeys } from '@/entities/user';
import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { onboardingStateSchema, type OnboardingState, type StepBodies, type StepName } from '../model/state';

/** Query-key factory (CLAUDE.md §8). */
export const onboardingKeys = {
  all: ['onboarding-v2'] as const,
  state: () => [...onboardingKeys.all, 'state'] as const,
  cycleSettings: () => [...onboardingKeys.all, 'cycle-settings'] as const,
};

const parse = (raw: unknown): OnboardingState => onboardingStateSchema.parse(raw);

/** GET /onboarding — the answers so far, to prefill each screen. */
export function useOnboardingState() {
  return useQuery({
    queryKey: onboardingKeys.state(),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/onboarding');
      return parse(data.data);
    },
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

type SaveStepInput = { [K in StepName]: { step: K; body: StepBodies[K] } }[StepName];

/** PUT /onboarding/steps/{step} — idempotent; the response is the new state. */
export function useSaveOnboardingStep() {
  const queryClient = useQueryClient();
  return useMutation<OnboardingState, unknown, SaveStepInput>({
    mutationFn: async ({ step, body }) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(`/onboarding/steps/${step}`, body);
      return parse(data.data);
    },
    onSuccess: (state) => {
      queryClient.setQueryData(onboardingKeys.state(), state);
      void queryClient.invalidateQueries({ queryKey: userKeys.all });
    },
  });
}

/** POST /onboarding/complete — «ورود به ریتمی»; 422 names the missing steps. */
export function useFinishOnboarding() {
  const queryClient = useQueryClient();
  return useMutation<OnboardingState, unknown, void>({
    mutationFn: async () => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/onboarding/complete');
      return parse(data.data);
    },
    onSuccess: (state) => {
      queryClient.setQueryData(onboardingKeys.state(), state);
    },
  });
}

interface BeforePeriodReminder {
  enabled: boolean;
  daysBefore: number;
}

const readReminder = (raw: unknown): BeforePeriodReminder | null => {
  const reminders = (raw as { reminders?: unknown } | null)?.reminders;
  if (!Array.isArray(reminders)) return null;
  const r = reminders.find((x: { code?: unknown }) => x?.code === 'before_period') as
    | { enabled?: unknown; days_before?: unknown }
    | undefined;
  if (!r) return null;
  return { enabled: r.enabled === true, daysBefore: typeof r.days_before === 'number' ? r.days_before : 2 };
};

/** The «یادآور قبل از پریود» switch of the Ready screen (GET /profile/cycle-settings). */
export function useBeforePeriodReminder(enabled: boolean) {
  return useQuery({
    queryKey: onboardingKeys.cycleSettings(),
    queryFn: async () => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/profile/cycle-settings');
      return readReminder(data.data);
    },
    enabled: enabled && isAuthenticated(),
    retry: 1,
  });
}

/** PUT /profile/cycle-settings with only the before-period switch. */
export function useSetBeforePeriodReminder() {
  const queryClient = useQueryClient();
  return useMutation<BeforePeriodReminder | null, unknown, boolean>({
    mutationFn: async (on) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>('/profile/cycle-settings', {
        reminders: { before_period: { enabled: on } },
      });
      return readReminder(data.data);
    },
    onSuccess: (saved) => {
      queryClient.setQueryData(onboardingKeys.cycleSettings(), saved);
    },
  });
}
