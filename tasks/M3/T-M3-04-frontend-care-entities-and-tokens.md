---
id: T-M3-04
title: Frontend — care-reminder entities, mutations, tokens, icons and i18n
milestone: M3
type: frontend
status: done
depends_on: []
parallel_group: M3-A
touches: [frontend/src/entities/care-reminder, frontend/src/features/manage-medication, frontend/src/features/manage-appointment, frontend/src/features/log-intake, frontend/src/app/globals.css, frontend/src/shared/ui/Icon.tsx, frontend/messages/fa, frontend/messages/en, frontend/src/shared/i18n, frontend/src/app/message-scopes.ts, frontend/next.config.ts]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test
---

# T-M3-04 — Frontend — care-reminder entities, mutations, tokens, icons and i18n

## Why
Foundation for every M3 screen. Can start before the backend: code against the contract in
docs/care-reminders/README.md (zod parsers + unit tests on fixture JSON).

## Scope
1. `entities/care-reminder`: types, zod boundary schemas and query hooks for medications, appointments, today,
   enums; a key factory (`careKeys`). Personal health data — never logged (§11).
2. Features: `manage-medication` (create/update/delete/toggle active), `manage-appointment` (create/update/delete/
   cancel/toggle prep item), `log-intake` (tick/untick a dose, **optimistic** update of `careKeys.today`).
   Every mutation invalidates through `careKeys`.
3. Tokens: map each color in the README table to existing tokens; add missing pairs to **both** `:root` and
   `[data-theme="dark"]` (e.g. success-soft, amber tile, rose soft/line) with the design's dark values.
4. Icons missing from `shared/ui/Icon.tsx`: capsule, tablet (pill exists), video, phone, mapPin, clock, bellRing,
   check, plus, user, note — stroke SVGs copied from the artboards.
5. i18n namespace `care` (fa + en) with all copy from the six artboards, registered in message scopes.
6. Local dev: `/api/v1/care/*` must reach backend-go while the rest goes to Laravel — add a dev-only rewrite/proxy
   (document the env var in frontend/CLAUDE.md or `.env.local` example). Stage/prod use nginx (T-M3-09).

## Out of scope
Any UI screen/widget (T-M3-05…08).

## Acceptance
- Parsers tested against README fixtures (incl. unknown enum values → safe fallback).
- `lint:dark` and `lint:styles` green with the new tokens; no hex outside `globals.css`.
