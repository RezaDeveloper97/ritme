import { aiDenialOf, isGateDenial, type AiGateDenial } from '@/entities/ai-credit';
import { ApiError } from '@/shared/api';

/*
 * What a failed `POST /labs` means for the upload screen. The AI gate refusals come from the ONE normaliser
 * `aiDenialOf` (entities/ai-credit):
 *
 * - `gate` — 402 ai_subscription_required / ai_credits_exhausted, 403 ai_feature_disabled → the gate sheet
 *   (`ai.handleError` opens it; the form shows nothing of its own);
 * - `consent` — 403 consent_required {consent, version} → the consent sheet;
 * - `validation` — 422 with field errors (files / category / taken_on);
 * - `busy` — any other 429 (ai_busy, ai_user_budget_exhausted, too_many_requests, the daily upload cap);
 * - `unavailable` — 503 (ai_budget_exhausted, storage off);
 * - `network` — no response (offline, timeout); `unknown` — anything else.
 *
 * Reads only the envelope's top-level fields — never a health payload.
 */
export type UploadErrorKind = 'gate' | 'consent' | 'validation' | 'busy' | 'unavailable' | 'network' | 'unknown';

export interface UploadError {
  kind: UploadErrorKind;
  /** The server's localized message, when it sent one. */
  message: string | null;
  /** 422 field → first message. */
  fields: Record<string, string>;
  denial: AiGateDenial | null;
}

function body(error: ApiError): Record<string, unknown> {
  const data = error.response?.data;
  return typeof data === 'object' && data !== null ? (data as Record<string, unknown>) : {};
}

export function uploadErrorOf(error: unknown): UploadError {
  const base: UploadError = { kind: 'unknown', message: null, fields: {}, denial: null };
  if (!(error instanceof ApiError)) return base;
  if (!error.response) return { ...base, kind: 'network' };
  const b = body(error);
  const message = typeof b.message === 'string' && b.message.trim() ? b.message : null;
  const status = error.response.status;
  const denial = aiDenialOf(error);
  if (isGateDenial(denial)) return { ...base, kind: 'gate', message, denial };
  if (denial?.kind === 'consent') return { ...base, kind: 'consent', message };
  if (status === 422) {
    const fields: Record<string, string> = {};
    const errs = b.errors;
    if (typeof errs === 'object' && errs !== null) {
      for (const [k, v] of Object.entries(errs as Record<string, unknown>)) {
        const first = Array.isArray(v) ? v.find((m) => typeof m === 'string') : undefined;
        if (typeof first === 'string') fields[k] = first;
      }
    }
    return { ...base, kind: 'validation', message, fields };
  }
  if (status === 429) return { ...base, kind: 'busy', message };
  if (status === 503) return { ...base, kind: 'unavailable', message };
  return { ...base, message };
}
