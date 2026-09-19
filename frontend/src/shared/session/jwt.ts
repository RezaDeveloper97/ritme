/**
 * Reads a JWT's `exp` claim locally, so the app knows when a refresh is due
 * without asking the server on every start.
 *
 * The signature is NOT checked — this only schedules a refresh; the server
 * stays the authority on whether the token is valid. Never log the token or its
 * payload (CLAUDE.md §11).
 */

function decodeBase64Url(segment: string): string | null {
  try {
    const base64 = segment.replace(/-/g, '+').replace(/_/g, '/');
    const padded = base64 + '='.repeat((4 - (base64.length % 4)) % 4);
    const binary = atob(padded);
    const bytes = Uint8Array.from(binary, (c) => c.charCodeAt(0));
    return new TextDecoder().decode(bytes);
  } catch {
    return null;
  }
}

/** Expiry of the token in epoch **milliseconds**, or null when unreadable. */
export function tokenExpiresAt(token: string): number | null {
  const [, payload] = token.split('.');
  if (!payload) return null;
  const json = decodeBase64Url(payload);
  if (!json) return null;
  try {
    const exp = (JSON.parse(json) as { exp?: unknown }).exp;
    return typeof exp === 'number' && Number.isFinite(exp) ? exp * 1000 : null;
  } catch {
    return null;
  }
}
