---
id: T-M10-12
title: Admin-web message rule editor v6
milestone: M10
type: frontend
status: todo
depends_on: [T-M10-03]
parallel_group: M10-W
touches: [admin-web/src/screens/message-rules, admin-web/src/features/rule-conditions, admin-web/src/entities/admin, admin-web/messages/fa.json, admin-web/messages/en.json]
skills: []
verify: cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# T-M10-12 — Admin-web message rule editor v6
**Model:** sonnet for the UI; the main session reviews the permission gating and the clinical-approval UI.

## Why
The editor must express the v6 model (T-M10-02, T-M10-03): typed facts from the registry, all_of / any_of / exclude,
N weighted variants, several caps, topics, datetime schedules with several hour windows, goal events, the approved vs
draft version, and the rule that only the clinical capability authors safety/clinical/content (gap 03 ADM-01).

## Scope
1. A reusable condition builder in `features/rule-conditions` (T-M10-13 reuses it for challenges): fact picker from
   `GET /api/admin/v1/messages/facts` with typed value inputs (number/interval, enum multi-select, boolean, date) and
   the operators valid for that type; all_of / any_of / exclude sections mapped to the stored groups.
2. Rule form: kind/level/slot with the T-M10-02 matrices (invalid combinations disabled with a reason), N variants
   with weights (sum shown, at most 6) and per-variant content, caps (day/week/cycle) + cooldown, topics (the
   `fertile_window` chip is required when a fertility fact is used), schedule (start/end datetime, several hour
   windows, "user-local time" note), goal event picker, spec names shown (type/slot/name, M-01).
3. Approval banner: "live: approved v{n} · draft v{m} awaiting approval" with a text diff between the two versions;
   approve/reject only for the clinical capability.
4. Authority in the UI mirrors the API: without the clinical capability, safety/clinical/content kinds are read-only;
   locked rules hide switch-off and delete, and only the clinical capability edits their copy.
5. 422 error codes from the API mapped to the fields; fa/en strings.

## Tests
- Vitest: form ↔ payload round trip for a legacy a/b rule (unchanged unless edited) and a full v6 rule; the condition
  builder emits the right groups; permission gating per capability; error mapping.
- Headless screenshots light and dark of the editor with a v6 rule and the approval banner (reviewed by the main
  session).

## Out of scope
- Simulator, inbox view, challenges, CTR (T-M10-13).

## Acceptance
- Verify passes; screenshots saved under `docs/engine-v6/qa/admin/`.
