---
id: T-M10-27
title: Postpartum analysis hub (backend endpoint and the existing screen)
milestone: M10
type: fullstack
status: done
depends_on: [T-M10-26, T-M10-22, T-M9-28, T-M9-06]
parallel_group: M10-D
touches: [backend-go/internal/i18n/testdata,backend-go/internal/analysis/postpartum_test.go,backend-go/internal/analysis/postpartum_int_test.go,backend-go/internal/analysis/postpartum.go, backend-go/internal/analysis/postpartum_handlers.go, backend-go/internal/http/routes_analysis_postpartum.go, backend-go/api/openapi.yaml, frontend/src/screens/analysis-postpartum, frontend/messages/fa/analysis-postpartum.json, frontend/messages/en/analysis-postpartum.json, backend-go/resources/translations/fa/analysis-postpartum.json, backend-go/resources/translations/en/analysis-postpartum.json]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./internal/analysis/... && make test-int PKG=./internal/analysis/... && cd ../frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:dark && npm run test && npm run copy:check -- analysis-postpartum
---

# T-M10-27 — Postpartum analysis hub (backend endpoint and the existing screen)
**Model:** opus (EPDS and bleeding are safety-relevant); sonnet for the screen.

## Why
SPEC §4.1 An_Hub_Post: EPDS trend, bleeding decline, breastfeeding, mother and baby sleep, child growth, weight. There
is no postpartum analysis in `internal/analysis`; `screens/analysis-postpartum` reads only `/postpartum/epds`
(gap 06 row 4.1k). DECISIONS.md A-10; D-176.

## Scope
1. `GET /api/v1/analysis/postpartum` (Go-only): EPDS (latest score and band; trend with ≥ 2), bleeding decline over
   days since delivery (recovery logs), breastfeeding (feeds per day from baby logs; the T-M9-28 flag), mother sleep
   (health log) and baby sleep (baby log), child growth (latest percentiles and the T-M10-22 crossing hint), weight
   versus pre-pregnancy. Each card has `ready` and `days_needed` (T-M10-26 rule).
2. Gates: postpartum and analysis module switches; outside postpartum → 409 `not_postpartum`; the EPDS card is never
   Plus-gated (SPEC §4.3); the other cards follow the "Plus cards per mode" rule of §4.3.
3. Frontend: `screens/analysis-postpartum` uses the endpoint; on 404 (Laravel) it falls back to today's EPDS-only call.

## Tests
- Fixtures: delivery 6 weeks ago with EPDS × 2, lochia logs, feeds, sleep, child measurements → every card ready.
- EPDS with one score → no trend; free user → EPDS card present, Plus cards locked.
- Not postpartum → 409; module off → 404/disabled per the modules convention.
- Frontend vitest: mapping and the 404 fallback; screenshots light/dark.

## Out of scope
- New postpartum logging (T-M9-06, T-M9-28).

## Acceptance
- Verify passes; PROGRESS.md cites D-176.

## Note from T-M10-26
EPDS minimums (1 score to show, 2 for a trend) are documented in `internal/analysis/params.go` `P` but not applied; apply them here.
