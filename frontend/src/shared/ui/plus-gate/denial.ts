import { ApiError } from '@/shared/api';

/**
 * Why the API refused a Plus feature (B-N2-06 gate, `internal/plus/gate.go`):
 *
 * - `locked` — 402 `plus_required`, reason `locked`: the feature is not in the
 *   user's tier. Upgrading opens it.
 * - `quota` — 402 `plus_required`, reason `quota`: a free allowance is spent
 *   (assistant 5/month). Upgrading lifts the cap.
 * - `exhausted` — 429 `plus_quota_exceeded`: a Plus/trial user spent a monthly
 *   quota (AI lab analysis 10/month). Upgrading does NOT help; it renews at
 *   `resetsAt`.
 */
export type PlusDenialKind = 'locked' | 'quota' | 'exhausted';

export interface PlusDenial {
  kind: PlusDenialKind;
  /** Entitlement key, e.g. `plus.lab_ai`. */
  feature: string;
  /** The monthly allowance that ran out (null for `locked`). */
  limit: number | null;
  /** ISO time the allowance renews (null for `locked`). */
  resetsAt: string | null;
  /** The server's localized message, when it sent one. */
  message: string | null;
}

export const PLUS_REQUIRED = 'plus_required';
export const PLUS_QUOTA_EXCEEDED = 'plus_quota_exceeded';

function str(v: unknown): string | null {
  return typeof v === 'string' && v.trim() ? v : null;
}

/**
 * Maps a failed API call to a {@link PlusDenial}, or null when the failure is
 * anything else (a 429 write throttle, a 402 from elsewhere, a network error).
 * Reads only the envelope's top-level gate fields — never a health payload.
 */
export function plusDenialOf(error: unknown): PlusDenial | null {
  if (!(error instanceof ApiError)) return null;
  const status = error.response?.status;
  const body = error.response?.data;
  if (typeof body !== 'object' || body === null) return null;
  const b = body as Record<string, unknown>;
  const code = b.error_code;
  const isRequired = status === 402 && code === PLUS_REQUIRED;
  const isExhausted = status === 429 && code === PLUS_QUOTA_EXCEEDED;
  if (!isRequired && !isExhausted) return null;
  const limit = typeof b.limit === 'number' && Number.isFinite(b.limit) ? b.limit : null;
  return {
    kind: isExhausted ? 'exhausted' : b.reason === 'quota' ? 'quota' : 'locked',
    feature: str(b.feature) ?? '',
    limit,
    resetsAt: str(b.resets_at),
    message: str(b.message),
  };
}

/** Whether upgrading to Plus would lift the denial (a paywall link makes sense). */
export function upgradeHelps(denial: PlusDenial): boolean {
  return denial.kind !== 'exhausted';
}
