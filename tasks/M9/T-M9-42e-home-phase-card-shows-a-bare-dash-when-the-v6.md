---
id: T-M9-42e
title: Home: phase card shows a bare dash when the v6 phase is null (pill / unknown day)
milestone: M9
type: frontend
status: todo
depends_on: [T-M9-42d,T-M9-40c]
parallel_group: M9-C
touches: [frontend/src/screens/home,frontend/messages/fa/home.json,frontend/messages/en/home.json,backend-go/resources/translations/fa/home.json,backend-go/resources/translations/en/home.json]
skills: [persian-writing]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run copy:check -- home
---

# T-M9-42e — Home: phase card shows a bare dash when the v6 phase is null (pill / unknown day)

## Why
Found in T-M9-42d QA (`docs/engine-v6/qa/T-M9-42d/today_unknown_ring_pill_*.png`, combined-pill persona,
`main_phase=unknown`): the daily phase card header renders only a dot and «—» where the phase name goes. It reads as
broken.

## Scope
- When the v6 phase is null/unknown, give the card a calm neutral header (e.g. a day/cycle label or "Today"), not a
  dash; for hormonal contraception prefer wording that doesn't imply a natural phase. Persian via `persian-writing`.
- Keep the Laravel path unchanged.

## Out of scope
- Backend.

## Acceptance
- Screenshot light + dark of the pill persona showing the new header (main session reviews copy); verify green.
