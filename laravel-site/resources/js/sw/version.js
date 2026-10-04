/**
 * Build ids and the two-tier update decision (L8-02). Shared by the service worker, the `pwa` data-module and
 * `node --test tests/js`.
 *
 * A build id is `YYYYMMDDHHmmss-<git sha>` (UTC build time, written by vite.config.js). Only the 14-digit time
 * orders builds — the sha is informational — so the admin may also enter a bare timestamp as the minimum.
 */
const STAMP = /^(\d{14})(?:-[0-9a-z]+)?$/i;

/** The sortable 14-digit stamp of a build id, or null when the value is not a build id (e.g. "dev"). */
export function buildStamp(id) {
    if (typeof id !== 'string') return null;
    const match = STAMP.exec(id.trim());

    return match ? match[1] : null;
}

/**
 * -1 / 0 / 1 like a comparator. Unknown or malformed ids compare as equal (0): an unparsable value must never
 * trigger an update, let alone the blocking screen.
 */
export function compareBuildIds(a, b) {
    const left = buildStamp(a);
    const right = buildStamp(b);
    if (left === null || right === null || left === right) return 0;

    return left < right ? -1 : 1; // fixed-width digit strings sort like numbers
}

/**
 * The update tier for the running bundle given `/pwa/version.json`:
 *  - "forced" — the running build is older than `min_build_id` (blocking screen),
 *  - "soft"   — a newer `build_id` is deployed (non-blocking toast),
 *  - "none"   — up to date, or nothing comparable.
 */
export function updateTier(current, info) {
    if (!info || typeof info !== 'object') return 'none';
    if (compareBuildIds(current, info.min_build_id) < 0) return 'forced';
    if (compareBuildIds(current, info.build_id) < 0) return 'soft';

    return 'none';
}
