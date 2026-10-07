// Public, build-time config (NEXT_PUBLIC_* is inlined into the bundle). Relative
// defaults = same origin: nginx on the instructor host proxies /api/ to backend-go,
// and `next dev` does the same through INSTRUCTOR_API_PROXY_TARGET.
const trim = (value: string | undefined, fallback: string): string =>
  (value?.trim() || fallback).replace(/\/+$/, '');

/** The shared user API (`/auth/send-otp`, `/auth/verify-otp`, `/auth/logout`). */
export const PUBLIC_API_BASE_URL = trim(process.env.NEXT_PUBLIC_API_BASE_URL, '/api/v1');

/** The instructor-scoped API (B-N8-01). */
export const INSTRUCTOR_API_BASE_URL = trim(
  process.env.NEXT_PUBLIC_INSTRUCTOR_API_BASE_URL,
  '/api/instructor/v1',
);
