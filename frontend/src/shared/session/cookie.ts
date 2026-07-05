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
