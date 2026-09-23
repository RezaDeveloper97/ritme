/**
 * Cookie names set by the Go admin API (docs/go-migration/admin-api.md §4).
 * With ADMIN_COOKIE_SECURE (production) they carry the `__Host-` prefix; plain
 * http local dev drops it. We accept both so the same build works everywhere.
 */
export const SESSION_COOKIE_NAMES = ['__Host-ritme_admin_session', 'ritme_admin_session'] as const;
export const CSRF_COOKIE_NAMES = ['__Host-ritme_admin_csrf', 'ritme_admin_csrf'] as const;
