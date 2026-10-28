'use client';

import { type QueryClient, useMutation, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient, ApiError, invalidateCycleDependents } from '@/shared/api';

/** Why POST /cycle/skips refused (422 `code`, T-M9-29) or failed without an answer. */
export type SkipErrorKind =
  | 'not_expected_yet'
  | 'period_started'
  | 'prediction_mismatch'
  | 'network'
  | 'other';

const SKIP_CODES: readonly string[] = ['not_expected_yet', 'period_started', 'prediction_mismatch'];

/** Maps a failed skip request to one of the calm messages. */
export function skipErrorKind(error: unknown): SkipErrorKind {
  if (!(error instanceof ApiError)) return 'other';
  if (!error.response) return 'network';
  const body = error.response.data;
  const code = typeof body === 'object' && body !== null ? (body as { code?: unknown }).code : undefined;
  if (error.response.status === 422 && typeof code === 'string' && SKIP_CODES.includes(code)) {
    return code as SkipErrorKind;
  }
  return 'other';
}

/**
 * POST /cycle/skips `{expected_start}` — «پریود نداشتم». The body only echoes the date until the endpoint returns
 * the today cycle_view (T-M9-32), so success refetches every cycle-dependent query (T-M9-41). A 422 that means
 * the prediction moved (or a period began) also refetches, so the screen shows the fresh state.
 */
export function skipMutationOptions(queryClient: QueryClient) {
  const refresh = () => invalidateCycleDependents(queryClient);
  return {
    mutationFn: async (expectedStart: string): Promise<string> => {
      const { data } = await apiClient.post<ApiEnvelope<{ expected_start: string }>>('/cycle/skips', {
        expected_start: expectedStart,
      });
      return data.data?.expected_start ?? expectedStart;
    },
    onSuccess: refresh,
    onError: (error: unknown) => {
      const kind = skipErrorKind(error);
      if (kind === 'prediction_mismatch' || kind === 'period_started') refresh();
    },
  };
}

/** DELETE /cycle/skips/{date}. A 404 means it is already gone, which is the state the user asked for. */
export function undoMutationOptions(queryClient: QueryClient) {
  return {
    mutationFn: async (expectedStart: string): Promise<void> => {
      try {
        await apiClient.delete(`/cycle/skips/${expectedStart}`);
      } catch (error) {
        if (error instanceof ApiError && error.response?.status === 404) return;
        throw error;
      }
    },
    onSuccess: () => invalidateCycleDependents(queryClient),
  };
}

export function useSkipPeriod() {
  const queryClient = useQueryClient();
  return useMutation(skipMutationOptions(queryClient));
}

export function useUndoSkip() {
  const queryClient = useQueryClient();
  return useMutation(undoMutationOptions(queryClient));
}
