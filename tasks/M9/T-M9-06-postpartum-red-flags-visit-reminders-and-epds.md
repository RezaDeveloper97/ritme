---
id: T-M9-06
title: Postpartum red flags, visit reminders and EPDS urgency as config
milestone: M9
type: fullstack
status: todo
depends_on: []
parallel_group: M9-A
touches: [backend-go/internal/postpartum, backend-go/internal/clinicalconfig, backend-go/internal/safety, backend-go/db/migrations, backend/database/migrations, backend-go/db/queries/postpartum, backend-go/api/openapi.yaml, frontend/src/screens/postpartum-recovery, frontend/src/screens/postpartum, frontend/src/entities/postpartum, frontend/messages/fa/postpartum.json, frontend/messages/en/postpartum.json]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && make schema-diff && go vet ./... && go test ./internal/postpartum/... ./internal/clinicalconfig/... ./internal/safety/... && make test-int PKG="./internal/postpartum/... ./db/migrations/..." && cd ../frontend && npm run typecheck && npm run lint && npm run lint:dark && npm run test
---

# T-M9-06 — Postpartum red flags, visit reminders and EPDS urgency as config
**Model:** opus (clinical safety). The main session reviews thresholds and copy. Runs alone in a batch (migration).

## Why
SPEC v6 §3.1: postpartum red flags (soaking a pad an hour, fever ≥ 38 °C, severe pain, breathlessness) are urgent,
and postpartum visits are due (contact ≤ 3 weeks, comprehensive ≤ 12 weeks, default reminder at 6 weeks). Today
`Recovery.Alerts()` raises only heavy lochia and large clots (`internal/postpartum/recovery.go:244-253`) and no visit
reminder exists anywhere (gap 04 rows 1.12, 1.13). EPDS total ≥ 13 currently also opens the crisis page
(`epds.go:147-150`); the spec reserves that for Q10 > 0 — a reduction, so it becomes a config value (H-01).

## Scope
1. Migration (goose + Laravel twin): on the postpartum recovery log table (find it in the postpartum migration) add
   `temperature_c DECIMAL(3,1) NULL`, `pad_soaked_within_hour TINYINT(1) NULL`, `severe_pain TINYINT(1) NULL`,
   `breathless TINYINT(1) NULL`.
2. Registry keys: `postpartum.redflags {fever_c 38.0}`, `postpartum.visits {contact_by_day 21, reminder_day 42,
   comprehensive_by_day 84}`, `epds.urgent_total {score 13}` (13 = today's behaviour; the advisor may set 31 = off).
3. `Recovery.Alerts()`: temperature ≥ fever_c, a pad soaked within an hour, severe pain or breathlessness → urgent
   with «همین حالا با ۱۱۵ تماس بگیر یا به اورژانس برو» and 115 — plus the existing lochia/clots alerts.
4. EPDS: Urgent when Q10 > 0 (unchanged) or total ≥ `epds.urgent_total`.
5. Visits: the postpartum home payload gains `visits: [{kind: contact|reminder|comprehensive, due_on, done}]`
   computed from the delivery date; a visit counts as done when a care appointment tagged postpartum (or a postpartum
   visit log, whichever exists — investigate) falls in the window. On or after `reminder_day` without a logged visit, one
   in-app notice is created lazily when the user opens the app (idempotent per kind). OpenAPI updated.
6. Frontend: the recovery sheet gets the four inputs (fa/en, light/dark, RTL), the urgent card with 115 appears when a
   red flag is saved; the postpartum home shows a «ویزیت‌های بعد از زایمان» card with the three dates and status.
7. Safety scenarios: `postpartum_fever`, `postpartum_pad_per_hour`, `postpartum_breathless`, `epds_q10_low_total`
   (Q10 = 1, total 4 → urgent with 115 and 123); `epds_high_total_urgent` reads the config.

## Tests
- 37.9 → none; 38.0 → urgent; pad soaked within an hour → urgent; severe pain → urgent; breathless → urgent; heavy
  lochia unchanged.
- EPDS: Q10 = 1, total 4 → urgent + 115 + 123; total 9 → low; 10 → possible, not urgent; 13 → likely + urgent (default);
  with `urgent_total = 31`: 13 → likely, not urgent; 30 with Q10 = 0 → not urgent.
- Visits for delivery 2026-10-01: contact by 2026-10-22, reminder 2026-11-12, comprehensive by 2026-12-24; a logged
  visit marks it done; the notice is created once.
- Migration up/down; schema-diff green; frontend vitest for the sheet mapper and the visits card.

## Out of scope
- Postpartum cycle return and breastfeeding (T-M9-28), the postpartum analysis hub (T-M10-27).

## Acceptance
- Tests green; verify passes; PROGRESS.md cites D-135 and lists H-01 as advisor-pending.
- Ask the user whether to deploy this fix to stage and beta now.
