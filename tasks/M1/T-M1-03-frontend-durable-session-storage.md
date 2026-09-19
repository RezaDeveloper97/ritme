---
id: T-M1-03
title: Frontend — durable session storage and safe 401 handling
milestone: M1
type: frontend
status: todo
depends_on: [T-M1-01]
parallel_group: M1-B
touches: [frontend/src/shared/session, frontend/src/shared/api/apiClient.ts, frontend/src/features/auth]
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

## Out of scope
Backend changes (T-M1-02), Android shell (T-M1-04).

## Acceptance
- Vitest units for: 401-classification, refresh single-flight, SessionGuard flag/token reconciliation.
- Manual check with the `local-dev` skill: sign in, delete the `ritme_auth` cookie, reload → still signed in;
  simulate an offline API → still signed in.
- typecheck, lint, fsd:lint, test green.
