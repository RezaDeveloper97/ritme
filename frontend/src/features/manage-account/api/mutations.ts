'use client';

import {
  type MutationOptions,
  type QueryClient,
  useMutation,
  useQueryClient,
} from '@tanstack/react-query';

import { type ApiEnvelope, apiClient, stepUpHeaders } from '@/shared/api';
import { loadPdfGenerator, shareOrDownloadFile } from '@/shared/lib/pdf';
import { clearAuthToken } from '@/shared/session';

import { type StepUpController, useStepUp } from '../model/useStepUp';
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
 * GET /profile/export with the single-use step-up token (T-M12-06, D-213): the
 * full export needs a fresh SMS code, so a stolen session alone cannot pull it.
 */
async function fetchExport(stepUpToken: string): Promise<Record<string, unknown>> {
  const { data } = await apiClient.get<ApiEnvelope<Record<string, unknown>>>('/profile/export', {
    headers: stepUpHeaders(stepUpToken),
  });
  return data.data ?? {};
}

/**
 * GET /profile/export — downloads the user's full personal data as a JSON file
 * on the client (Blob + object URL). The shape is passed through opaquely; this
 * code never reads individual health fields.
 */
async function exportProfileData(stepUpToken: string): Promise<void> {
  const payload = await fetchExport(stepUpToken);
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

interface ExportAction<TArgs extends unknown[]> {
  /** Opens the step-up sheet; the export starts once the code is confirmed. */
  exportData: (...args: TArgs) => void;
  isPending: boolean;
  isError: boolean;
  /** Spread onto `<StepUpSheet {...x.stepUp} />` next to the trigger. */
  stepUp: StepUpController;
}

/** Export-my-data action with a pending flag for the triggering row/button. */
export function useExportData(): ExportAction<[]> {
  const step = useStepUp();
  const mutation = useMutation<void, unknown, string>({
    mutationFn: exportProfileData,
  });
  return {
    exportData: () => step.request((token) => mutation.mutate(token)),
    isPending: mutation.isPending,
    isError: mutation.isError,
    stepUp: step.sheet,
  };
}

/**
 * The same export as a PDF (B-N1-12, «JSON و PDF»): GET /profile/export,
 * rendered to a paper document on the device by `shared/lib/pdf` and handed to
 * the share sheet / a download. Nothing is uploaded or logged; the labels come
 * from the caller (this slice owns no copy).
 */
export function useExportPdf(): Omit<ExportAction<[PdfExportLabels]>, 'exportData'> & {
  exportPdf: (labels: PdfExportLabels) => void;
} {
  const step = useStepUp();
  const mutation = useMutation<void, unknown, { labels: PdfExportLabels; token: string }>({
    mutationFn: async ({ labels, token }) => {
      const payload = await fetchExport(token);
      const { renderPdf } = await loadPdfGenerator();
      const blob = await renderPdf(exportPdfSpec(payload, labels));
      await shareOrDownloadFile(blob, EXPORT_PDF_FILENAME);
    },
  });
  return {
    exportPdf: (labels) => step.request((token) => mutation.mutate({ labels, token })),
    isPending: mutation.isPending,
    isError: mutation.isError,
    stepUp: step.sheet,
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
): MutationOptions<void, unknown, string> {
  return {
    // The step-up token (T-M12-06): deletion needs a fresh SMS code too.
    mutationFn: async (stepUpToken) => {
      await apiClient.delete<ApiEnvelope<never>>('/account', { headers: stepUpHeaders(stepUpToken) });
    },
    onSuccess: () => {
      clearAuthToken();
      queryClient.clear();
    },
  };
}

export function useDeleteAccount() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, string>(deleteAccountMutationOptions(queryClient));
}
