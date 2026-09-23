---
id: T-M5-03
title: Fertility — GET /fertility/insights (window, confidence, evidence, history)
milestone: M5
type: backend
status: todo
depends_on: [T-M5-02]
parallel_group: M5-C
touches: [backend-go/internal/fertility, backend-go/db/queries/fertility, backend-go/internal/http/routes_fertility.go, backend-go/api/openapi.yaml]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/fertility/... && make test-int PKG=./internal/fertility/...
---

# T-M5-03 — Fertility — GET /fertility/insights (window, confidence, evidence, history)

## Why
Artboard `v19_TTC_Insights` explains where the prediction comes from.

## Scope
1. Window + ovulation from the existing cycle view (no new prediction model); `cycles_used`.
2. Evidence rows: complete cycles (typical length ± variability → strength), BBT shifts in recent cycles (days),
   LH tests this cycle (positive/none); strength `strong|medium|none`; confidence `low|medium|high` derived from
   them (document the rule in the README).
3. History: last 5 cycles' estimated ovulation day (confirmed BBT shift day, else positive LH + 1, else engine
   estimate) with Jalali/Gregorian month labels.
4. Tips list (localized) driven by what is missing (log BBT daily, LH from day N, …).
5. Tests with fixtures for 0, 2 and 6 cycles.

## Acceptance
- Shape per README; tests green.
