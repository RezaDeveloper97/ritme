/**
 * Public runtime config. NEXT_PUBLIC_* values are inlined at build time, so
 * they are public by definition — never put a secret here.
 */

function trimSlash(url: string): string {
  return url.replace(/\/+$/, '');
}

/** Go admin API base (`/api/admin/v1`). Same-origin in production. */
export const ADMIN_API_BASE_URL = trimSlash(
  process.env.NEXT_PUBLIC_ADMIN_API_BASE_URL || '/api/admin/v1',
);

/** Public API base, used for GET /languages (the content-language registry). */
export const PUBLIC_API_BASE_URL = trimSlash(process.env.NEXT_PUBLIC_API_BASE_URL || '/api/v1');
