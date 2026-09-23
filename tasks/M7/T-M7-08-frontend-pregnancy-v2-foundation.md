---
id: T-M7-08
title: Frontend — pregnancy v2 entity/features, tokens (light+dark), illustrations, icons, i18n
milestone: M7
type: frontend
status: done
depends_on: []
parallel_group: M7-A
touches: [frontend/src/entities/pregnancy, frontend/src/features/track-pregnancy, frontend/src/shared/ui/illustrations, frontend/src/app/globals.css, frontend/src/shared/ui/Icon.tsx, frontend/messages/fa, frontend/messages/en, frontend/src/shared/i18n, frontend/src/app/message-scopes.ts]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test
---

# T-M7-08 — Frontend — pregnancy v2 foundation

## Scope
1. `entities/pregnancy`: v2 zod schemas + queries (`pregnancyKeys.v2.*`: today, week, day, calendar, alerts,
   datingPreview, report) alongside v1 (keep v1 working until screens switch).
2. `features/track-pregnancy`: v2 mutations (day log PUT, week state, alert actions) with invalidation.
3. Tokens: map the design palette per docs/pregnancy-v2/README.md (no dark artboard → derive from existing dark tokens; add pairs only when
   needed, both themes); `lint:dark` green.
4. `shared/ui/illustrations`: WelcomePregnancy, ResultBaby, FetusSize(`illustrationKey`) as token-coloured SVG
   components (keys match the admin picker).
5. Icons from the artboards (hand, heart, eye/face, drop, scale, bookmark, face moods, warning triangle, doctor).
6. i18n namespace `pregnancyV2` fa+en (static UI copy only — content comes from the API).

## Acceptance
- Parser tests on fixtures; gates green.
