/**
 * The auth-flag cookie name. It lives in its own module — deliberately WITHOUT
 * a `'use client'` directive — so it is a plain string on both sides of the
 * boundary. `token.ts` is a client module; if this constant were exported from
 * there, importing it into the (server/edge) middleware would yield a
 * client-reference proxy instead of the string, and every auth check would
 * silently fail. Keep server-readable constants out of client modules.
 *
 * The cookie is a value-less flag only — never the token itself (CLAUDE.md §11).
 */
export const AUTH_COOKIE = 'ritme_auth';

/**
 * Marks a session whose signup onboarding is unfinished. Its value is the step
 * key to resume at (see `entities/user/model/steps`), so a visitor who drops out
 * mid-flow — or comes back on a later visit — re-enters exactly where they
 * stopped instead of landing in the app half-registered.
 *
 * It is a separate cookie from {@link AUTH_COOKIE} because the two answer
 * different questions: the auth flag says the mobile number was verified, this
 * one says registration was actually finished. Verifying an OTP grants the
 * first immediately, which is why the second is needed at all.
 */
export const ONBOARDING_COOKIE = 'ritme_onboarding';

/**
 * Auth screens reachable without a session. Lives here (not in the middleware)
 * so the client-side session guard gates on exactly the same list the edge does
 * — two copies would drift and produce redirect loops.
 */
export const PUBLIC_SEGMENTS = ['splash', 'welcome', 'signup', 'otp'] as const;

/** Fired on the window when the session is dropped, so the UI can leave guarded screens. */
export const SESSION_CLEARED_EVENT = 'ritme:session-cleared';
