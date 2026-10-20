---
id: T-M10-19
title: Vitals completion: glucose contexts, diabetic targets, report minimums, copy from the registry
milestone: M10
type: fullstack
status: todo
depends_on: [T-M10-16, T-M10-17]
parallel_group: M10-C
touches: [backend-go/internal/vitals/thresholds.go, backend-go/internal/vitals/reports.go, backend-go/internal/vitals/model.go, backend-go/internal/vitals/handlers.go, backend-go/internal/vitals/service.go, backend-go/internal/vitals/targets.go, backend-go/internal/vitals/lang, backend-go/internal/http/routes_vitals.go, backend-go/api/openapi.yaml, backend-go/contract/allowlist, frontend/src/screens/vitals-hub, frontend/src/screens/vitals-report, frontend/src/screens/vitals-add, frontend/src/entities/vital, frontend/messages/fa/vitals.json, frontend/messages/en/vitals.json, backend-go/resources/translations/fa/vitals.json, backend-go/resources/translations/en/vitals.json]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./internal/vitals/... && make test-int PKG=./internal/vitals/... && make contract ROUTES=vitals && cd ../frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:dark && npm run test && npm run copy:check -- vitals
---

# T-M10-19 — Vitals completion: glucose contexts, diabetic targets, report minimums, copy from the registry
**Model:** opus (clinical classification). The main session reviews thresholds and copy.

## Why
Gap 04 rows 2.5, 2.6, 2.11, 2.2: glucometer post-meal, bedtime and random readings share the diagnostic 140/200 bands;
there are no diabetic targets or doctor overrides; reports average and classify a single reading (SPEC §3.2 and §4.1:
at least 2 readings on 2 days); the crisis copy hard-codes «۱۸۰/۱۲۰» and «۵۴», so it goes stale when the registry
changes. Pregnancy-aware classification already shipped in T-M9-05. DECISIONS.md H-09; D-170.

## Scope
1. Glucose contexts behind `glucose.contexts.spec` (H-09, default 0 = today): with 1, glucometer post-meal < 140 is in
   range and ≥ 140 is `discuss` (no 200 class); random and bedtime have no class below 200 and `very_high` from 200
   with the "if you have symptoms, see a doctor" copy. Fasting, hypo (< 70, < 54) and pregnancy (T-M9-05) unchanged;
   hypo always wins.
2. Diabetic targets: when `chronic_illnesses` contains `diabetes`, fasting/pre-meal readings are classified against
   `glucose.diabetic` (80–130) and post-meal against < 180 (`below_target | in_target | above_target`), unless the
   user's `vital_targets` row for that context overrides it. `GET/PUT/DELETE /api/v1/vitals/targets` (owner only,
   values within physiological bounds, `set_by` = doctor|user). Non-diabetic users never see target classes.
3. Reports (`reports.go:120-135,248-290`): mean, min–max and category distribution only with ≥ 2 readings on ≥ 2
   different local days in the period; otherwise `not_enough_data` with `readings_needed`.
4. Copy: vitals alert texts (lang files and the `vitals_alert` `message_contents` rows) take `{systolic}`,
   `{diastolic}`, `{mgdl}` placeholders filled from the registry version of the reading's date; a test fails when a
   vitals text contains a hard-coded threshold number.
5. Frontend: «داده کافی نیست؛ N اندازه‌گیری دیگر» state on the hub and report; a targets editor for diabetic users
   («هدف‌های قندی که پزشکت تعیین کرده»); classes and labels from the API; fa/en, light/dark.

## Tests
- SPEC §6.1 BP: 119/79 normal; 120/79 elevated; 129/79 elevated; 130/79 stage1; 129/80 stage1; 139/89 stage1;
  140/89 stage2; 180/120 stage2; 181/100 crisis; 150/121 crisis.
- Glucose fasting 99 / 100 / 125 / 126; 70 / 69 low; 54 low; 53 urgent.
- `spec 0`: post-meal 139/140/199/200 as today; `spec 1`: post-meal 139 in range, 140 discuss, 250 discuss; random 199
  no class, 200 very_high; bedtime same as random.
- Diabetic: fasting 79 below, 80 in, 130 in, 131 above; post-meal 179 in, 180 above; override 90–140 honoured; a
  non-diabetic user with a targets row is classified as non-diabetic.
- Reports: 2 readings on 1 day → not_enough_data; 2 readings on 2 days → stats.
- Copy test (no hard-coded thresholds; placeholders filled after a registry change).
- Frontend vitest for the not-enough-data state and the targets form.

## Out of scope
- HbA1c (a lab, T-M10-20), pregnancy tables (T-M9-05), the analysis vitals card minimum (T-M10-26).

## Acceptance
- Verify passes; PROGRESS.md cites D-170 and notes H-09 as advisor-pending.
