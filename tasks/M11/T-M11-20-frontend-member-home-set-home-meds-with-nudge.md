---
id: T-M11-20
title: Frontend member home set: home, meds with nudge and propose, calendar with attendance, alert, postpartum
milestone: M11
type: frontend
status: blocked
depends_on: [T-M11-10, T-M11-12, T-M11-13, T-M11-14]
parallel_group: M11-W
touches: [frontend/src/screens/companion-home, frontend/src/screens/family-meds, frontend/src/screens/family-calendar, frontend/src/screens/family-alert, frontend/src/screens/family-postpartum, frontend/src/entities/family, frontend/src/app/[locale]/companion, frontend/src/app/[locale]/family, frontend/src/app/message-scopes.ts, frontend/messages/fa/family.json, frontend/messages/en/family.json, frontend/messages/fa/companion-home.json, frontend/messages/en/companion-home.json, backend-go/resources/translations/fa/family.json, backend-go/resources/translations/en/family.json, backend-go/resources/translations/fa/companion-home.json, backend-go/resources/translations/en/companion-home.json]
skills: []
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run copy:check -- family companion-home
---

# T-M11-20 — Frontend member home set: home, meds with nudge and propose, calendar with attendance, alert, postpartum
**Model:** sonnet; the main session reviews what each screen reveals about the owner.

## Why
SPEC §1.8 Fam_PHome, Fam_PMeds, Fam_PCalendar, Fam_PAlert and Fam_PPostpartum are the member's screens; today the
companion home is male-only and has no meds actions, calendar, alert or postpartum view. DECISIONS.md F-10, F-22,
F-24.

## Scope
1. Home (`/companion`, every member — remove the client-side gender gate): per-owner cards from `/family/home`
   (summary, today's family tasks, appointments with «من هم می‌آیم», support tip, household list, children).
2. Meds (`/family/meds`): per person; actions by level — nudge (with the refusal reasons `quiet_hours`, `not_due`,
   quota), log a dose for a child (guardian), propose a medication (202 → sent for approval), personal reminder link.
3. Calendar (`/family/calendar`): shared calendar with filters; attendance; add an appointment — direct for a child,
   proposal for the owner, personal for herself.
4. Alert (`/family/alerts`): neutral title, time, call her, 115, help guide, history; seen on open.
5. Postpartum (`/family/postpartum`): night shift one tap, feed logging (pumped milk as feed type `pump`), her
   postpartum visit (appointments view), the general «حالش را بپرس» card (static copy, no data).
6. Fallback on 404 (Laravel) to today's companion home. Light and dark; fa/en.

## Tests
- Vitest: level-dependent actions, nudge refusal mapping, attendance toggle, proposal 202 path, alert payload with
  no extra fields rendered; headless screenshots light and dark under `docs/engine-v6/qa/m11/`.

## Out of scope
- Backend.

## Acceptance
- Verify passes; screenshots reviewed by the main session.

## Note from T-M11-06 (2026-10-29)
`frontend/src/entities/companion/api/companion-side.ts` still parses the pre-v6 section shapes (`has_data`,
`days_to_period`, symptoms). Since T-M11-06 the server sends the v6 member DTOs (`cycle_phase`: `main_phase`,
`period_soon` partner-only, no numbers; symptoms gone → 404); the `.catch` hides it but the cycle card stays empty.
Move the parser to the v6 DTOs (OpenAPI `Family*` schemas) here.

## Blocked
Suspended by the user on 2026-10-31: all family (M11) work is on hold. Do not pick, plan or run this task until the user lifts the suspension explicitly.
