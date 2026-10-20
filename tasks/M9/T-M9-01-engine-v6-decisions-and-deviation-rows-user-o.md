---
id: T-M9-01
title: Engine v6 decisions and deviation rows (user OK)
milestone: M9
type: investigate
status: done
depends_on: []
parallel_group: M9-A
touches: [docs/engine-v6/DECISIONS.md, docs/go-migration/deviations.md]
skills: []
verify: test "$(awk '/^## E /,/^## M /' docs/engine-v6/DECISIONS.md | grep -c 'Status:\*\* proposed')" = 0 && grep -q '^| D-139 ' docs/go-migration/deviations.md
---

# T-M9-01 — Engine v6 decisions and deviation rows (user OK)
**Model:** main session (opus). These are product decisions; do not delegate.

## Why
SPEC v6 leaves about 45 engine, input and cross-cutting questions open (`docs/engine-v6/DECISIONS.md` sections E, I,
X: e.g. how the calendar shows a late period, what the engine outputs under the pill or after delivery, what a skipped
cycle means, whether the onboarding period stays painted as logged). The engine library (T-M9-15…22), the schema
(T-M9-24) and every serving task implement these answers; changing them after the code lands is rework. Evidence:
`docs/engine-v6/gap/01-cycle-core.md` §3, `02-inputs-consumers.md` §2–§4, `07-infra-performance.md` §7.

## Scope
1. Re-read DECISIONS.md E/I/X and the cited gap sections; check nothing in the code changed since 2026-10-21 that
   makes a default wrong (spot-check the file:line refs of I-01, I-02, E-13, E-22).
2. Ask the user in Persian, in two groups, one line per decision with the recommended default:
   - **Visible product changes:** I-01 (onboarding period becomes "assumed", not painted as logged), E-04 (late
     chain), E-13 (pill), E-14 (postpartum), E-15 (skip), E-22 (history layout), I-02 (bleeding logs make real
     periods), I-05 (postpartum flow from day 42), X-03 (stage → beta rollout), X-05 (legacy keys).
   - **Technical defaults:** everything else in E, I, X.
   Offer "OK to all defaults" and accept per-item changes. Wait for the answer.
3. Record each answer in DECISIONS.md: `Status: accepted` or `changed — <what>`; the approval date at the top.
4. Write rows **D-120 … D-139** (DECISIONS.md X-08 table) in `docs/go-migration/deviations.md`, same columns as the
   existing rows (Laravel behaviour / Go behaviour / Status `accepted — engine v6, not shipped yet`). Do not delete
   D-30 or D-55; change their status text to point at D-121/D-122 and D-124.
5. If an answer changes a task's scope, edit that task file (Scope/Tests/Acceptance) now and list the edits in
   `tasks/PROGRESS.md`.

## Out of scope
- Code. Message (M), health (H), analysis (A) and family (F) decisions — T-M10-01 and T-M11-01.

## Acceptance
- No E/I/X decision is `proposed`; the user's approval date is recorded.
- D-120…D-139 exist; D-30 and D-55 point at their successors.
- Every task whose scope changed is edited and listed in PROGRESS.md.
