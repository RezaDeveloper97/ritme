import { getApiErrorStatus } from '@/shared/api';

/**
 * Maps a failed auth request to an i18n key suffix under `auth.errors`, so the
 * UI shows a calm, localized Persian message instead of the raw English server
 * string. Health data is never involved here, but keep messages non-alarming
 * (CLAUDE.md §11).
 */
export type AuthErrorKey = 'invalidCode' | 'tooMany' | 'network' | 'generic';

export function authErrorKey(error: unknown): AuthErrorKey {
  const status = getApiErrorStatus(error);
  if (status === undefined) return 'network';
  if (status === 429) return 'tooMany';
  if (status === 400 || status === 422) return 'invalidCode';
  return 'generic';
}
