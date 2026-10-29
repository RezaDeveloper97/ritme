---
id: T-M10-26
title: Analysis methods per SPEC 4.1: minimums, correlation strength, mood by phase, top finding, lab trend
milestone: M10
type: fullstack
status: done
depends_on: [T-M10-01, T-M9-35]
parallel_group: M10-D
touches: [backend-go/internal/analysis, backend-go/internal/i18n/testdata, backend-go/internal/http/routes_analysis.go, backend-go/api/openapi.yaml, backend-go/contract/allowlist, backend-go/contract/golden-v6, frontend/src/screens/analysis, frontend/src/screens/analysis-correlations, frontend/src/screens/analysis-symptoms, frontend/src/screens/analysis-body, frontend/src/screens/analysis-monthly, frontend/src/screens/analysis-labs, frontend/src/screens/analysis-ttc, frontend/src/entities/analysis, frontend/messages/fa/analysis.json, frontend/messages/en/analysis.json, backend-go/resources/translations/fa/analysis.json, backend-go/resources/translations/en/analysis.json]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/analysis/... && make test-int PKG=./internal/analysis/... && make contract ROUTES=analysis,analysis-ttc,analysis-pregnancy && cd ../frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:dark && npm run test && npm run copy:check -- analysis
---

# T-M10-26 — Analysis methods per SPEC 4.1: minimums, correlation strength, mood by phase, top finding, lab trend
**Model:** opus for the methods; sonnet for the screens; the main session reviews every "finding" text.

## Why
SPEC §4.1 fixes each analysis's method and minimum data. The code differs (gap 06 §2.1): correlations need 20 paired
days and use Cohen φ strengths (spec: 30 days and a ratio ≥ 2× is strong); mood by phase is Cramér's V with 5 days per
group (spec: % of days per mood in each phase, ≥ 10 days per phase); weight is ready at 2 points (spec: 3 logs in 14
days); BP/glucose use a generic minimum (spec: 2 logs on 2 days); BBT needs 9 readings (spec: ≥ 10 in the cycle);
the monthly report needs 7 days (spec: 10); the top finding is a fixed priority list (spec: strength × recency); the
"not enough data" nudge counts cycles (spec: «N روز دیگر ثبت کن»); there is no per-marker lab trend. T-M9-35 already
moved regularity to the engine. DECISIONS.md A-01, A-02, A-12; D-175.

## Scope
1. `analysis.Params` (A-12) — one documented struct: length 2 cycles; regularity 3 cycles within 12 months (engine
   verdict); period intensity 2 periods; symptom pattern ≥ 3 cycles; mood by phase ≥ 10 days per phase;
   correlations 30 paired days, strong = ≥ 30 days and ratio ≥ 2×; weight 3 logs in 14 days; BP/glucose 2 logs on 2
   days; lab trend 2 labs per marker; BBT ≥ 10 temperatures in the cycle (`engine/bbt`); pregnancy weight BMI + 2 logs;
   EPDS 1 score, trend with 2; monthly report 10 logged days.
2. Correlations: strength from the group-rate ratio (A-02); φ kept as an extra field; copy «هم‌زمانی در داده‌های تو، نه
   علت» on every correlation (verify the exact string).
3. Mood by phase: % of days with each mood in each phase (only phases with ≥ 10 logged days).
4. Weight: MA7, Δ7, Δ30 when 3 logs fall within the last 14 days; BP/glucose: mean, min–max, category distribution
   with 2 logs on 2 days (categories from the vitals classifier, pregnancy-aware).
5. Lab trend (An_Labs): `GET /api/v1/analysis/labs?marker=` — the same marker across consecutive reports (confirmed
   values only, T-M10-20), ≥ 2 reports; Plus (SPEC §4.3 «روند آزمایش»).
6. Top finding: score = strength (0–1 per analysis, documented table) × recency (half-life 30 days on the finding's
   latest evidence day); ties by a fixed analysis order; the hub shows the highest.
7. Nudge: «برای دیدن الگوها N روز دیگر ثبت کن» with N in days from the nearest unmet minimum (a cycle minimum is
   converted with the effective cycle length, rounded up).
8. TTC guidance: under 35 after 12 months and 35+ after 6 months of trying without success → «با پزشک مشورت کن»; add it
   if missing.
9. Frontend: the screens read `strength`, `days_needed` and the mood-by-phase shares; fa/en, light/dark.
10. Contract: Laravel goldens stay for v1; v6 Go-native goldens re-recorded; differences allow-listed under D-175.

## Tests
- Boundaries: correlations at 29/30 paired days and ratio 1.9/2.0; mood 9/10 days in a phase; weight 2 vs 3 logs in
  14 days and 3 logs over 15 days; BP 2 logs on 1 day vs 2 days; BBT 9/10; monthly 9/10 days; lab trend 1/2 reports;
  regularity with 3 cycles inside vs outside 12 months.
- Top finding: an older strong finding vs a recent medium one (expected winner from the formula).
- Nudge days for a correlation with 18 paired days → 12.
- Frontend vitest for the new fields.

## Out of scope
- Caching (T-M10-28), the postpartum hub (T-M10-27), Plus quota changes (T-M10-29).

## Acceptance
- Verify passes; PROGRESS.md cites D-175.
