---
id: T-M2-16
title: Pregnancy engine and all /pregnancy endpoints
milestone: M2
type: backend
status: done
depends_on: [T-M2-05, T-M2-06, T-M2-07, T-M2-08]
parallel_group: M2-E
touches: [backend-go/internal/pregnancy, backend-go/internal/http/routes_pregnancy.go, backend-go/db/queries/pregnancy, backend-go/contract/allowlist/pregnancy.yaml]
skills: []
verify: cd backend-go && go test ./internal/pregnancy/... && make test-int PKG=./internal/pregnancy/... && make contract ROUTES=pregnancy
---

# T-M2-16 — Pregnancy engine and all /pregnancy endpoints

## Why
Pregnancy mode is a full feature with 26 routes, its own calculation and alert rules. Read api-inventory §1.8,
domain-inventory §1 (pregnancy tables), §2 (plain `date` cast on PregnancyProfile!), §4.2, and the three
`Pregnancy*Controller`s + FormRequests.

## Scope
1. `pregnancy/calc` = `PregnancyCalculationService` (GA from LMP/ultrasound/manual, confidence ± days, trimester, EDD
   +280, conception, `getCurrentWeek` clamp with null→1, fetal movement thresholds 18/24, high-risk rules,
   Gregorian-with-Persian-month-names `formatPersianDate`, raw `{en,fa}` blobs in `formatted`/`flags`, Carbon float
   `diffInDays` for `days_remaining`/`total_days`).
2. `pregnancy/alerts` = `PregnancyAlertService` (all rules, titles stored in request locale, `medical_history_flags`
   `[]` vs object, sugar values as decimal strings in `trigger_symptoms`).
3. `pregnancy/content`: weekly content with the "whole object when locale missing" behaviour; `/content/{week}` 1–40
   else 422; fa/en clamp default **en**.
4. All routes in api-inventory §1.8, with static-before-param order (`enums`, `summary`, `read-all`), 201 on
   onboarding/symptoms/weekly/fetal-movement POST, framework 422 family with the Persian FormRequest messages,
   `fetal_movement_felt` update, upsert by week.
5. Serializers for PregnancyProfile (plain `date` casts → previous-day UTC), symptom/weekly/fetal logs
   (`date:Y-m-d`, decimal strings, `time` columns), alerts.

## Out of scope
Pregnancy home sections and pregnancy messages (T-M2-17/18 consume `pregnancy/calc`).

## Acceptance
- `make contract ROUTES=pregnancy` green for all pregnant personas (LMP/ultrasound/manual, weeks 8/26) and the
  non-pregnant 404/400 paths, fa/en.
- Alert rule table tests mirroring each PHP rule.
