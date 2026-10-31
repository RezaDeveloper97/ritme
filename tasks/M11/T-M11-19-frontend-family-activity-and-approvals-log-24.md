---
id: T-M11-19
title: Frontend family activity and approvals: log, 24-hour undo, proposals and share requests
milestone: M11
type: frontend
status: blocked
depends_on: [T-M11-09, T-M11-11, T-M11-14]
parallel_group: M11-W
touches: [frontend/src/screens/family-activity, frontend/src/features/family-approvals, frontend/src/entities/family, frontend/src/app/[locale]/family/activity, frontend/src/app/message-scopes.ts, frontend/messages/fa/family.json, frontend/messages/en/family.json, backend-go/resources/translations/fa/family.json, backend-go/resources/translations/en/family.json]
skills: []
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run copy:check -- family
---

# T-M11-19 — Frontend family activity and approvals: log, 24-hour undo, proposals and share requests
**Model:** sonnet; the main session reviews the approval copy.

## Why
SPEC §1.8 Fam_Activity: the circle manager sees the activity log filtered by member, undoes within 24 hours and
approves or rejects proposals; the member sees what she proposed. No screen exists. DECISIONS.md F-14, F-22.

## Scope
1. `screens/family-activity` at `/family/activity` (new `family` namespace, registered in `message-scopes.ts`):
   activity list with member filter, undo button until the deadline (409 `changed_since` explained), pending
   proposals (approve/reject, the proposed item shown to the owner), share requests, child two-guardian requests.
2. Member view: «پیشنهادهای من» with status and cancel.
3. Entry points: hub (T-M11-17) and the owner's notice. Light and dark; fa/en.

## Tests
- Vitest: filters, undo availability by deadline, approve/reject flows, error mapping; headless screenshots light
  and dark under `docs/engine-v6/qa/m11/`.

## Out of scope
- Backend.

## Acceptance
- Verify passes; screenshots reviewed by the main session.

## Blocked
Suspended by the user on 2026-10-31: all family (M11) work is on hold. Do not pick, plan or run this task until the user lifts the suspension explicitly.
