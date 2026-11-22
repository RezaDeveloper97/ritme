---
id: T-M12-05
title: Purge offline downloads and the downloads-page cache on logout, delete and account switch
milestone: M12
type: frontend
status: done
depends_on: []
parallel_group: M12-SEC
touches: [frontend/src/shared/pwa/offline, frontend/src/shared/session/cleanup.ts, frontend/scripts/sw.template.js, frontend/src/entities/course/api/offline.ts]
skills: [pwa]
verify: cd frontend && npm run typecheck && npx vitest run src/shared/pwa src/shared/session
---

# T-M12-05 — Purge offline downloads and the downloads-page cache on logout, delete and account switch
**Model:** sonnet (opus reviews). Evidence: `docs/quality-2026-11/findings.md`.

## Why
S-09: on a shared or handed-down phone the next person sees and can play the previous user's downloaded lessons; titles like IVF or pregnancy loss reveal her health status.

## Scope
1. Register an `onSessionEnd` cleanup that deletes the `ritme-offline-media` IndexedDB, resets the cached connection and key, and messages the SW to delete `ritme-offline-pages-v1`.
2. Tag each stored item with the user id; on read, drop items whose id differs from the current session.
3. Never edit `public/sw.js` directly — change the template and regenerate (memory: PWA).

## Out of scope
- Re-designing offline downloads.
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- vitest with fake-indexeddb: save → `runSessionCleanups()` → list is empty; an item tagged with another user id is never returned.
- Manual check documented: download, log out, log in as another number → «دانلودها» empty.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
