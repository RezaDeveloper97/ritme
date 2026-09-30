'use client';

import {
  type MutationOptions,
  type QueryClient,
  useMutation,
  useQueryClient,
} from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { loadPdfGenerator, shareOrDownloadFile } from '@/shared/lib/pdf';
import { clearAuthToken } from '@/shared/session';

import { exportPdfSpec, type PdfExportLabels } from './export-pdf';

/**
 * Account data actions (CLAUDE.md §11 — export & delete are first-class user
 * rights). The export never touches logs or analytics: the payload goes
 * straight from the response into a local file download and is never inspected
 * or persisted here.
 */

const EXPORT_FILENAME = 'ritme-data-export.json';
const EXPORT_PDF_FILENAME = 'ritme-data-export.pdf';

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
 * The same export as a PDF (B-N1-12, «JSON و PDF»): GET /profile/export,
 * rendered to a paper document on the device by `shared/lib/pdf` and handed to
 * the share sheet / a download. Nothing is uploaded or logged; the labels come
 * from the caller (this slice owns no copy).
 */
export function useExportPdf() {
  const mutation = useMutation<void, unknown, PdfExportLabels>({
    mutationFn: async (labels) => {
      const { data } = await apiClient.get<ApiEnvelope<Record<string, unknown>>>('/profile/export');
      const { renderPdf } = await loadPdfGenerator();
      const blob = await renderPdf(exportPdfSpec(data.data ?? {}, labels));
      await shareOrDownloadFile(blob, EXPORT_PDF_FILENAME);
    },
  });
  return {
    exportPdf: mutation.mutate,
    isPending: mutation.isPending,
    isError: mutation.isError,
  };
}

/**
 * DELETE /account — permanently deletes the account and all its data and
 * revokes tokens server-side. On success the local session (token, flag
 * cookie, per-user device data via the session cleanups) and the entire query
 * cache are cleared so no health data lingers on the device or in memory.
 */
export function deleteAccountMutationOptions(
  queryClient: QueryClient,
): MutationOptions<void, unknown, void> {
  return {
    mutationFn: async () => {
      await apiClient.delete<ApiEnvelope<never>>('/account');
    },
    onSuccess: () => {
      clearAuthToken();
      queryClient.clear();
    },
  };
}

export function useDeleteAccount() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, void>(deleteAccountMutationOptions(queryClient));
}
