---
id: T-M11-21
title: Frontend child co-guardian UI: actor names, two-guardian approvals, carer logging, medication log
milestone: M11
type: frontend
status: todo
depends_on: [T-M11-11, T-M11-14]
parallel_group: M11-W
touches: [frontend/src/screens/children, frontend/src/screens/child-home, frontend/src/screens/child-add, frontend/src/screens/child-meds, frontend/src/entities/child, frontend/src/app/[locale]/children, frontend/src/app/message-scopes.ts, frontend/messages/fa/children.json, frontend/messages/en/children.json, backend-go/resources/translations/fa/children.json, backend-go/resources/translations/en/children.json]
skills: []
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run copy:check -- children
---

# T-M11-21 — Frontend child co-guardian UI: actor names, two-guardian approvals, carer logging, medication log
**Model:** sonnet.

## Why
SPEC §1.8 Fam_PChild: co-parents log feeds, diapers, sleep and medication, see vaccines, growth and milestones, and
every entry shows who logged it; deleting a child or removing a parent asks the other parent. The child screens are
owner-only today. DECISIONS.md F-15, F-26.

## Scope
1. Actor name on every entry (feeds, sleeps, diapers, measurements, vaccines, milestones, med doses); "—" when unknown.
2. Guardians section on the child home: list, invite a circle member, accept/decline a guardian invite, leave.
3. Delete child / remove guardian → «منتظر تأیید والد دیگر» with the pending request; the other parent approves from
   the activity screen (T-M11-19).
4. Carers see only feed, diaper and sleep logging.
5. `screens/child-meds`: the medication dose log (F-26) at `/children/[id]/meds`. Light and dark; fa/en.

## Tests
- Vitest: actor rendering, carer-limited actions, pending-approval state, med-dose form; headless screenshots light
  and dark under `docs/engine-v6/qa/m11/`.

## Out of scope
- Backend.

## Acceptance
- Verify passes; screenshots reviewed by the main session.
