import { aiDenialOf, isGateDenial } from '@/entities/ai-credit';
import { ApiError } from '@/shared/api';

/*
 * What a failed `POST /health-record/documents/{id}/extract` means for the document screen (AI gate order:
 * throttle → feature on + subscription + credits → consent ai_documents → cost / busy → reserve, CB-REC-02):
 *
 * - `gate` — 402 ai_subscription_required / ai_credits_exhausted, 403 ai_feature_disabled → the gate sheet
 *   (`useAiGate('doc_extract').handleError` opens it; the page shows nothing else);
 * - `consent` — 403 consent_required → the consent sheet, then retry;
 * - `noFiles` — 422 on `file_ids` (nothing to read);
 * - `busy` — any other 429 or a 503 (ai_busy, budgets, provider off);
 * - `conflict` — 409 (already running / confirmed) → just re-read the document;
 * - `unknown` — anything else (network included).
 *
 * Reads only the envelope's top-level codes — never a health payload.
 */
export type ExtractErrorKind = 'gate' | 'consent' | 'noFiles' | 'busy' | 'conflict' | 'unknown';

export function extractErrorOf(error: unknown): ExtractErrorKind {
  if (!(error instanceof ApiError) || !error.response) return 'unknown';
  const status = error.response.status;
  const denial = aiDenialOf(error);
  if (isGateDenial(denial)) return 'gate';
  if (denial?.kind === 'consent') return 'consent';
  if (status === 422) return 'noFiles';
  if (status === 409) return 'conflict';
  if (status === 429 || status === 503) return 'busy';
  return 'unknown';
}
