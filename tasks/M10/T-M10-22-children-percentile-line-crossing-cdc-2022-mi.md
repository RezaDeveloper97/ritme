---
id: T-M10-22
title: Children: percentile-line crossing, CDC 2022 milestone bands, versioned vaccine schedule
milestone: M10
type: fullstack
status: done
depends_on: [T-M10-16]
parallel_group: M10-C
touches: [backend-go/internal/children, backend-go/db/migrations, backend/database/migrations, backend-go/contract/fixtures/dump.sql, docs/engine-v6/clinical-signoff.md, backend-go/api/openapi.yaml, backend-go/contract/allowlist/children.yaml, admin-web/src/screens/child-content, frontend/src/screens/child-growth, frontend/src/screens/child-milestones, frontend/src/screens/child-vaccines, frontend/src/entities/child, frontend/messages/fa/children.json, frontend/messages/en/children.json, backend-go/resources/translations/fa/children.json, backend-go/resources/translations/en/children.json]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/children/... && make test-int PKG="./internal/children/... ./db/migrations/..." && make contract ROUTES=children && cd ../frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:dark && npm run test && npm run copy:check -- children
---

# T-M10-22 — Children: percentile-line crossing, CDC 2022 milestone bands, versioned vaccine schedule
**Model:** opus (child health rules); sonnet may draft the milestone copy, the main session reviews it.

## Why
Gap 04 rows 5.2–5.4: only the latest growth value is checked (no "crossing two major percentile lines" rule); the
milestone bands lack 15 and 30 months (CDC 2022) and include a 3-month band CDC does not have; the vaccine schedule is
an editable catalog with no version, so a schedule change rewrites every child's history. DECISIONS.md H-15, H-17;
D-173.

## Scope
1. Growth crossing: between consecutive measurements of the same metric (weight, length/height, head circumference),
   count the major lines (P3, P15, P50, P85, P97) crossed; `≥ child.growth.cross_lines` → `discuss` with copy
   («با پزشک کودک در میان بگذار»). The band rule (< P3, > P97) stays.
2. Milestones: data migration adding 15- and 30-month bands to the `child_milestones` catalog (plus activities and the
   doctor note where the other bands have them) from the CDC 2022 "Learn the Signs. Act Early." checklists, fa/en,
   `needs_review = 1`; the 3-month band stays, marked as Ritme's own (H-15) in `meta`.
3. Vaccines: due ages come from `child.vaccine_schedule` (T-M10-16) in the version in force on the child's birth date
   (H-17); the catalog keeps names and copy; dose records are unchanged; the admin child-content screen shows ages
   read-only with a link to clinical-config.
4. Frontend: the growth chart shows the crossing hint; milestone tabs include 15 and 30 months; fa/en, light/dark.

## Tests
- P2.9 discuss, P3.0 in band, P97.0 in band, P97.1 discuss.
- P60 → P10 (crosses 50 and 15) → discuss; P60 → P20 (crosses 50) → none; P16 → P14 → none.
- Schedule v2 effective 2027-01-01: a child born 2026-12-31 follows v1, one born 2027-01-01 follows v2.
- Milestone band lookup at 14, 15, 29, 30 months.
- Migration up/down.

## Out of scope
- Catch-up vaccination logic (stays manual, H-17).

## Acceptance
- Verify passes; PROGRESS.md cites D-173; new milestone copy listed in clinical-signoff.md for review.
