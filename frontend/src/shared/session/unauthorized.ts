/**
 * Does this failed response mean the session is over?
 *
 * Only a 401 that the API itself sent about *our* token ends a session. The
 * deployed bundle used to sign the user out on any 401, and a 401 can come from
 * places that say nothing about the token: a proxy's password gate (staging's
 * HTML challenge did exactly that), a request that never carried the token, or
 * a request that raced a token refresh. Signing out costs the user an OTP round
 * trip and a possible cooldown, so the rule is "clear only when told to".
 *
 * Pure and framework-free so it is unit-tested in `unauthorized.test.ts`.
 */

/**
 * `error_code` values (T-M1-02) that mean the token we sent can never work again.
 * `unauthenticated` is included: the checks above guarantee the request carried
 * our current token, so the API is saying that exact token is malformed, badly
 * signed (key rotation) or no longer exists (deleted user). Keeping it would
 * leave the app stuck on a dead token.
 */
export const SESSION_ENDING_CODES = ['token_expired', 'token_revoked', 'unauthenticated'] as const;

/** Laravel's stock body for `AuthenticationException` — the shape before `error_code` existed. */
const LEGACY_UNAUTHENTICATED_MESSAGE = 'Unauthenticated.';

export interface UnauthorizedResponse {
  status: number | undefined;
  contentType: string | undefined;
  body: unknown;
  /** The bearer token the failed request carried, if any. */
  sentToken: string | null;
  /** The token the app holds right now. */
  currentToken: string | null;
}

function field(body: unknown, key: string): unknown {
  return body !== null && typeof body === 'object' ? (body as Record<string, unknown>)[key] : undefined;
}

export function endsSession({
  status,
  contentType,
  body,
  sentToken,
  currentToken,
}: UnauthorizedResponse): boolean {
  if (status !== 401) return false;
  // Laravel always answers JSON; anything else is a proxy/gateway speaking.
  if (!String(contentType ?? '').toLowerCase().includes('json')) return false;
  // A request without our token tells us nothing about our token.
  if (!sentToken) return false;
  // The token changed while the request was in flight (refresh, re-sign-in):
  // the old one being rejected must not take the new one down with it.
  if (sentToken !== currentToken) return false;

  const code = field(body, 'error_code');
  if (typeof code === 'string') {
    return (SESSION_ENDING_CODES as readonly string[]).includes(code);
  }

  // A backend that predates `error_code` (prod until T-M1-02 ships): trust only
  // Laravel's exact unauthenticated body.
  return field(body, 'message') === LEGACY_UNAUTHENTICATED_MESSAGE;
}

/** The token in an `Authorization: Bearer …` header value, or null. */
export function bearerOf(header: unknown): string | null {
  if (typeof header !== 'string') return null;
  const match = /^Bearer\s+(.+)$/i.exec(header.trim());
  return match ? match[1] : null;
}
