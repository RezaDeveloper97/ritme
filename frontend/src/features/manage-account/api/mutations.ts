'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { clearAuthToken } from '@/shared/session';

/**
 * Account data actions (CLAUDE.md §11 — export & delete are first-class user
 * rights). The export never touches logs or analytics: the payload goes
 * straight from the response into a local file download and is never inspected
 * or persisted here.
 */

const EXPORT_FILENAME = 'ritme-data-export.json';

/**
 * GET /profile/export — downloads the user's full personal data as a JSON file
 * on the client (Blob + object URL). The shape is passed through opaquely; this
 * code never reads individual health fields.
 */
async function exportProfileData(): Promise<void> {
  const { data } = await apiClient.get<ApiEnvelope<Record<string, unknown>>>(
    '/profile/export',
  );

  const payload = data.data ?? {};
  const blob = new Blob([JSON.stringify(payload, null, 2)], {
    type: 'application/json',
  });
  const url = URL.createObjectURL(blob);
  try {
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = EXPORT_FILENAME;
    document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();
  } finally {
    URL.revokeObjectURL(url);
  }
}

/** Export-my-data action with a pending flag for the triggering row/button. */
export function useExportData() {
  const mutation = useMutation<void, unknown, void>({
    mutationFn: exportProfileData,
  });
  return {
    exportData: mutation.mutate,
    isPending: mutation.isPending,
    isError: mutation.isError,
  };
}

/**
 * DELETE /account — permanently deletes the account and all its data and
 * revokes tokens server-side. On success the local session and the entire
 * query cache are cleared so no health data lingers in memory.
 */
export function useDeleteAccount() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, void>({
    mutationFn: async () => {
      await apiClient.delete<ApiEnvelope<never>>('/account');
    },
    onSuccess: () => {
      clearAuthToken();
      queryClient.clear();
    },
  });
}
