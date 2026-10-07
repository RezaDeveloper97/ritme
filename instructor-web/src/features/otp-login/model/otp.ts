import { isApiError } from '@/shared/api';

/** The OTP the API issues: 4 digits, resend after 60 s (backend-go internal/auth). */
export const OTP_LENGTH = 4;
export const RESEND_AFTER_SECONDS = 60;

/** Message key (namespace `login.errors`) for a failed send-otp. */
export function sendErrorKey(error: unknown): 'invalidMobile' | 'wait' | 'network' | 'generic' {
  if (!isApiError(error)) return 'generic';
  if (error.code === 'network_error') return 'network';
  if (error.status === 422) return 'invalidMobile';
  if (error.status === 429) return 'wait';
  return 'generic';
}

/** Message key (namespace `login.errors`) for a failed verify-otp. */
export function verifyErrorKey(
  error: unknown,
): 'wrongCode' | 'expired' | 'tooManyAttempts' | 'blocked' | 'network' | 'generic' {
  if (!isApiError(error)) return 'generic';
  if (error.code === 'network_error') return 'network';
  switch (error.status) {
    case 422:
      return 'wrongCode';
    case 400:
      return 'expired';
    case 429:
      return 'tooManyAttempts';
    case 403:
      return 'blocked';
    default:
      return 'generic';
  }
}

/** Seconds to wait from a 429 (body `data.retry_after`), clamped to [1, 120]. */
export function retryAfterSeconds(error: unknown): number {
  const raw = isApiError(error) ? error.retryAfter : null;
  if (raw === null || !Number.isFinite(raw)) return RESEND_AFTER_SECONDS;
  return Math.min(120, Math.max(1, Math.ceil(raw)));
}
