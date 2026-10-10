---
id: CB-COND-07b
title: Home condition nudge card
epic: COND
type: frontend
status: todo
depends_on: [CB-COND-07]
parallel_group: COND-E
touches: [frontend/src/screens/home,frontend/src/entities/condition]
skills: [new-fsd-slice]
boards: [nbl_Cond_Hub.dc.html]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-COND-07b — Home condition nudge card

## Why
CB-COND-02 showed the condition nudge only on `/programs`. Bloom's home has no slot for it (found in CB-COND-07),
so a user with heavy bleeding or pain days never sees `GET /messages/nudges` unless she opens the hub.

## Boards
`nbl_Cond_Hub.dc.html` (the nudge card as already built in screens/conditions).

## Scope
1. A soft nudge card on the cycle-mode home (reuse the hub's card and the `entities/condition` nudge query), shown only when `/messages/nudges` returns a hit; the action link opens the program, the card disappears once she enrols.
2. Light + dark, fa + en shots; unit test for the show/hide rule.

## Out of scope
- Backend (route already mounted by CB-COND-06b); dismissing nudges; Android.

## Acceptance
- Home shows the card for a not-enrolled user with 2+ heavy-flow days, and nothing after enrolment.
- `verify` green.
