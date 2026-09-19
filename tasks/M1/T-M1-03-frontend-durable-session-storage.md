---
id: T-M1-03
title: Frontend — durable session storage and safe 401 handling
milestone: M1
type: frontend
status: in_progress
depends_on: [T-M1-01]
parallel_group: M1-B
touches: [frontend/src/shared/session, frontend/src/shared/api/apiClient.ts, frontend/src/features/auth, frontend/src/app/[locale], frontend/src/screens/auth-*, frontend/src/shared/pwa/InstallPrompt.tsx]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run test
---

# T-M1-03 — Frontend — durable session storage and safe 401 handling

## Why
Fix the client-side part of the root cause from `docs/investigations/session-logout.md`.

## Scope
- Only clear the session on a 401 whose body says the token is gone (`error_code` from T-M1-02, or the Laravel
  `Unauthenticated.` shape) — never on network errors, 5xx, timeouts, or a 401 from a non-auth path.
- Call `navigator.storage.persist()` once after sign-in (where supported) so the browser doesn't evict storage.
- Sliding refresh: when T-M1-02's endpoint exists, refresh on app start/resume if the token is within its window
  (decode `exp` locally; no extra request when it's far away). Single-flight — concurrent requests must not
  trigger parallel refreshes.
- `SessionGuard`: never drop a present token because the flag cookie is missing (restore the flag instead —
  already the intent; add tests that lock it in, including the resume/visibility path).
- Keep CLAUDE.md §11: the token never goes into a cookie value, URL or log.

## Scope change from T-M1-01 (docs/investigations/session-logout.md) — this task carries the main fix
Root cause #1: a missing `ritme_auth` flag (WebKit caps `document.cookie` at 7 days; other kinds of cookie loss) makes
the middleware render `/signup`. `SessionGuard` restores the flag but **leaves the user on the sign-in form**, even
though a valid token is still in `localStorage`. Root cause #2: iOS storage partitions (Safari tab / Home Screen app /
in-app browser), confirmed in prod logs. Add to scope:
- If a token exists when `/splash`, `/welcome`, `/signup` or `/otp` mounts, restore the flag and `router.replace` to
  `/home` (or to the pending onboarding step). Lock this in with a test.
- Set the flag with an HTTP `Set-Cookie` from a same-origin Next route handler (a server-set cookie is not subject to
  the WebKit 7-day cap), and re-assert it on every start and resume, not only when it is missing.
- **Prod still runs the any-401 bundle** (v1.0.2, built 2026-09-01). The JSON-only guard exists only in the repo and
  staging, so this task must ship through T-M1-14.
- iOS UX: the install hint says the Home Screen app needs one sign-in. Optional "open in Safari" hint for
  Instagram/Telegram in-app browsers.
- Remove or implement the dead `ritme_onboarded` check in `app/[locale]/page.tsx`.
- Manual check addition: on an iPhone, sign in → Add to Home Screen → open it (expect the sign-in screen once; after
  that it must stay signed in for days).
- Also touches: `frontend/src/app/[locale]` (auth screens / route handler), `frontend/src/screens/auth-*`,
  `frontend/src/shared/pwa/InstallPrompt.tsx`.

## Out of scope
Backend changes (T-M1-02), Android shell (T-M1-04).

## Acceptance
- Vitest units for: 401-classification, refresh single-flight, SessionGuard flag/token reconciliation.
- Manual check with the `local-dev` skill: sign in, delete the `ritme_auth` cookie, reload → still signed in;
  simulate an offline API → still signed in.
- typecheck, lint, fsd:lint, test green.
