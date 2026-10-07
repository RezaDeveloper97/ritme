import { ApiError } from '@/shared/api';

/*
 * What a failed `POST /health-record/documents/{id}/extract` means for the document screen (AI gate order:
 * throttle → plus.doc_ai → consent ai_documents → cost / busy → reserve, CB-REC-02):
 *
 * - `plus` — 402 plus_required (free users) / 429 plus_quota_exceeded → the manual form + the Plus link;
 * - `consent` — 403 consent_required → the consent sheet, then retry;
 * - `noFiles` — 422 on `file_ids` (nothing to read);
 * - `busy` — any other 429 or a 503 (ai_busy, budgets, provider off);
 * - `conflict` — 409 (already running / confirmed) → just re-read the document;
 * - `unknown` — anything else (network included).
 *
 * Reads only the envelope's top-level codes — never a health payload.
 */
export type ExtractErrorKind = 'plus' | 'consent' | 'noFiles' | 'busy' | 'conflict' | 'unknown';

export function extractErrorOf(error: unknown): ExtractErrorKind {
  if (!(error instanceof ApiError) || !error.response) return 'unknown';
  const status = error.response.status;
  const data = error.response.data;
  const code = typeof data === 'object' && data !== null ? (data as Record<string, unknown>).error_code : undefined;
  if (status === 402 || code === 'plus_required' || code === 'plus_quota_exceeded') return 'plus';
  if (status === 403 && code === 'consent_required') return 'consent';
  if (status === 422) return 'noFiles';
  if (status === 409) return 'conflict';
  if (status === 429 || status === 503) return 'busy';
  return 'unknown';
}
