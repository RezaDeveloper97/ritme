/**
 * The running app version, stamped into the bundle at build time from
 * package.json (next.config.ts `NEXT_PUBLIC_APP_VERSION`, the same value
 * scripts/generate-version.mjs writes to /version.json). `0.0.0` only when
 * the env is missing (unit tests).
 */
export const APP_VERSION: string = process.env.NEXT_PUBLIC_APP_VERSION ?? '0.0.0';
