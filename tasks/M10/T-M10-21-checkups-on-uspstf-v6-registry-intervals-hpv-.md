---
id: T-M10-21
title: Checkups on USPSTF v6: registry intervals, HPV by last test, breast awareness, follow-up
milestone: M10
type: fullstack
status: done
depends_on: [T-M10-16, T-M10-17]
parallel_group: M10-C
touches: [backend-go/internal/checkups, backend-go/internal/admin/checkups, backend-go/db/migrations, backend-go/contract/allowlist/checkups.yaml, admin-web/src/screens/checkup-types, frontend/src/screens/checkups, frontend/src/screens/checkup-detail, frontend/src/screens/checkup-mark-done, frontend/src/screens/checkup-self-exam, frontend/src/entities/checkup, frontend/messages/fa/checkups.json, frontend/messages/en/checkups.json, backend-go/resources/translations/fa/checkups.json, backend-go/resources/translations/en/checkups.json]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/checkups/... && make test-int PKG="./internal/checkups/... ./db/migrations/..." && cd ../frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:dark && npm run test && npm run copy:check -- checkups && cd ../admin-web && npm run typecheck && npm run test
---

# T-M10-21 — Checkups on USPSTF v6: registry intervals, HPV by last test, breast awareness, follow-up
**Model:** opus (screening rules). The main session reviews intervals and copy; the catalog stays in the clinical
review doc.

## Why
Gap 04 rows 4.1–4.5: Pap every 36 months for 21–65 with no HPV option; mammography 12–24 months from 40 with no upper
age; no BP or glucose screening; the monthly breast self-exam is still due (v6: breast awareness, no due date); a
"needs follow-up" result does not stop the default interval; non-urgent screenings are hidden in pregnancy instead of
labelled "after pregnancy". DECISIONS.md H-10, H-11, H-16; D-172.

## Scope
1. The engine switches from the catalog columns to `screening.uspstf` (T-M10-16; the SPEC §3.4 values the user
   approved in T-M10-01 and listed in clinical-signoff.md) for the cervical, mammography, BP, glucose and dental types.
2. Catalog data migration (`checkup_types`): add `bp_screening` and `glucose_screening` with fa/en copy (why, prep,
   performed by), `source_note` citing USPSTF; `breast_self_exam` becomes `informational = 1` «آگاهی از سینه» (H-10;
   no due date, history read-only); the Pap item's copy mentions the HPV option; extra types per H-16 unchanged.
3. Engine (`checkups/engine`): registry-driven types read `screening.uspstf` rows on the evaluation day (age band, last
   record's `test_kind`, BMI from the profile via `bmi.who`); other types keep the catalog columns; a last record with
   `result = follow_up` sets no default next due (only an explicit `next_due_on`); informational types never have a
   due date or reminder; in pregnancy, non-urgent items are returned with `label: after_pregnancy` and no reminders
   instead of being hidden.
4. Record API: `test_kind` accepted for the cervical type (required from age 30).
5. Admin-web checkup-types: interval/age fields of registry-driven types are read-only with a link to
   clinical-config; the informational toggle.
6. Frontend: informational card (no due date), «پیگیری با پزشک» state for follow-up, «بعد از بارداری» label, test-kind
   picker on the cervical record form; fa/en, light/dark.

## Tests
- Cervical: age 20 none, 21 due (cytology 36), 29 cytology, 30 with last cytology → 36, with last HPV → 60, 65 included,
  66 excluded.
- Mammography 39 / 40 / 74 / 75; glucose at 35 with BMI 24.9 vs 25.0, at 71 excluded; BP at 39 (36 months) vs 40
  (12 months).
- Next due = last + interval with end-of-month clamp (2026-01-31 + 1 month = 2026-02-28).
- follow_up → no default due; informational → no due date; pregnant → `after_pregnancy` and no reminder.
- Migration up/down; admin-web vitest for read-only fields.

## Out of scope
- New checkup content beyond the two new types.

## Acceptance
- Verify passes; PROGRESS.md cites D-172 and lists H-10, H-11, H-16 as advisor-pending.

## Batch note (main session, 2026-10-22)
`backend-go/api/openapi.yaml` was removed from touches so this task can run in parallel with T-M10-18. Write the needed OpenAPI change as a unified diff to the scratchpad and report its path; the main session applies it before the commit.
