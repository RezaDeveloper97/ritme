import { isApiError } from '@/shared/api';

/** The 422 / 409 / 403 answers of the rule API, mapped to the editor's fields and a banner. */
const BANNER_CODES: Record<string, string> = {
  locked: 'errLocked',
  approval_required: 'errApprovalRequired',
  not_pending: 'errNotPending',
  approval_not_needed: 'errApprovalNotNeeded',
  unknown_fact: 'errUnknownFact',
  bad_value: 'errBadValue',
};

/** i18n key (messageRules namespace) of an error's banner, or null when the message should be the server's. */
export function bannerKey(error: unknown): string | null {
  if (!isApiError(error)) return null;
  if (error.status === 403) return 'errForbiddenClinical';
  return BANNER_CODES[error.code] ?? null;
}

/** Editor field → condition builder: both lists show under the builder. */
const FIELD_ALIASES: Record<string, readonly string[]> = {
  conditions: ['conditions', 'exceptions'],
  caps: ['caps', 'cap_count', 'cap_period', 'cooldown_days'],
  hours: ['hour_windows', 'hours_from', 'hours_to'],
  variants: ['variants', 'content.variants', 'content.b', 'ab_split'],
  schedule: ['starts_at', 'ends_at', 'starts_on', 'ends_on'],
};

/** The first server message for a field (or a group of fields); `undefined` when there is none. */
export function fieldError(error: unknown, field: string): string | undefined {
  if (!isApiError(error)) return undefined;
  for (const name of FIELD_ALIASES[field] ?? [field]) {
    const m = error.field(name);
    if (m) return m;
  }
  return undefined;
}
