---
id: T-M10-26b
title: Analysis follow-ups: pregnancy weight minimum, menopause rate-ratio, top finding on TTC and pregnancy hubs
milestone: M10
type: fullstack
status: todo
depends_on: [T-M10-26]
parallel_group: M10-D
touches: [backend-go/internal/analysis,backend-go/internal/menopause,backend-go/contract/golden-v6,backend-go/contract/allowlist,frontend/src/screens/analysis-pregnancy,frontend/src/screens/analysis-ttc,frontend/messages/fa/analysis-pregnancy.json,frontend/messages/en/analysis-pregnancy.json,backend-go/resources/translations/fa/analysis-pregnancy.json,backend-go/resources/translations/en/analysis-pregnancy.json,backend-go/internal/i18n/testdata]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/analysis/... ./internal/menopause/... ./internal/i18n/... && make test-int PKG="./internal/analysis/... ./internal/menopause/..." && make contract ROUTES=analysis,analysis-ttc,analysis-pregnancy && cd ../frontend && npm run typecheck && npm run test && npm run fsd:lint
---

# T-M10-26b — Analysis follow-ups: pregnancy weight minimum, menopause rate-ratio, top finding on TTC and pregnancy hubs

## Why
Open items from T-M10-26 (SPEC §4.1, D-175) that sat outside its touches.

## Scope
1. Pregnancy weight: apply the `P` minimum (2 logs) in the pregnancy analysis; below it return `ready:false` +
   `logs_needed` and show the not-enough line (analysis-pregnancy copy, fa+en, written-warm «تو»).
2. Menopause `analysis.Binary` still uses the old φ / 20-day rule: move it to the shared rate-ratio method and the
   30-day minimum from `internal/analysis/params.go`.
3. TTC and pregnancy hubs: add the strength × recency top finding (same scorer as the cycle hub).
4. After any copy change: resync `resources/translations`, re-record `internal/i18n/testdata` (`go test
   ./internal/i18n -run MatchesLaravelGoldens -update`) and check `make contract ROUTES=public`.

## Acceptance
- Each item tested at the boundary; goldens re-recorded and explained; no causal wording; verify green.
