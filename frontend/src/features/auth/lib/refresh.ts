/**
 * Sliding session: a token close to expiry is traded for a fresh one-year token
 * (`POST /auth/refresh-session`, T-M1-02) on app start/resume, without an OTP.
 *
 * The expiry is read locally from the JWT, so a far-away expiry costs no request
 * at all. Refreshes are single-flight: however many callers ask at once (start
 * and resume can overlap), one request goes out and all of them await it — two
 * parallel refreshes would each revoke the other's token.
 *
 * Framework-free and dependency-injected so `refresh.test.ts` can drive it.
 */

const DAY_MS = 24 * 60 * 60 * 1000;

/** Matches the backend's `passport.refresh_window_days`. */
export const REFRESH_WINDOW_MS = 30 * DAY_MS;

/** A failed attempt isn't retried on every resume. */
export const RETRY_AFTER_MS = 10 * 60 * 1000;

/** Due when the token is still valid but inside the refresh window. */
export function isRefreshDue(expiresAt: number | null, now: number): boolean {
  if (expiresAt === null) return false;
  const left = expiresAt - now;
  // An expired token can't refresh itself; the next request's 401 ends it.
  return left > 0 && left < REFRESH_WINDOW_MS;
}

/** What one refresh request produced. */
export type RefreshOutcome =
  | { kind: 'refreshed'; token: string }
  /** Not due yet by the server's clock — nothing to do. */
  | { kind: 'unchanged' }
  /** The server has no refresh endpoint (older backend) — stop asking. */
  | { kind: 'unavailable' };

export interface RefresherDeps {
  getToken: () => string | null;
  setToken: (token: string) => void;
  expiresAt: (token: string) => number | null;
  /** Performs the request; may throw on network/HTTP errors. */
  request: () => Promise<RefreshOutcome>;
  now: () => number;
}

export interface SessionRefresher {
  /** Resolves once any refresh this call joined or started has settled. Never rejects. */
  refreshIfDue: () => Promise<void>;
}

export function createSessionRefresher(deps: RefresherDeps): SessionRefresher {
  let inFlight: Promise<void> | null = null;
  let unavailable = false;
  let lastAttemptAt = -Infinity;

  const run = async (sent: string): Promise<void> => {
    try {
      const outcome = await deps.request();
      if (outcome.kind === 'unavailable') {
        unavailable = true;
        return;
      }
      // Signed out or signed in again meanwhile: the answer is about a token
      // this device no longer holds.
      if (outcome.kind === 'refreshed' && deps.getToken() === sent) {
        deps.setToken(outcome.token);
      }
    } catch {
      // Offline, 5xx, timeout: keep the current token and try again later. A
      // real "token is gone" 401 is handled by the HTTP client's interceptor.
    }
  };

  return {
    refreshIfDue() {
      if (inFlight) return inFlight;
      if (unavailable) return Promise.resolve();

      const token = deps.getToken();
      const now = deps.now();
      if (!token || !isRefreshDue(deps.expiresAt(token), now)) return Promise.resolve();
      if (now - lastAttemptAt < RETRY_AFTER_MS) return Promise.resolve();

      lastAttemptAt = now;
      inFlight = run(token).finally(() => {
        inFlight = null;
      });
      return inFlight;
    },
  };
}
