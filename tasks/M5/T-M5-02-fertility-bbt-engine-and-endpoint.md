---
id: T-M5-02
title: Fertility — BBT shift engine (3-over-6) and GET /fertility/bbt
milestone: M5
type: backend
status: done
depends_on: [T-M5-01]
parallel_group: M5-B
touches: [backend-go/internal/fertility, backend-go/db/queries/fertility, backend-go/internal/http/routes_fertility.go, backend-go/api/openapi.yaml]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/fertility/... && make test-int PKG=./internal/fertility/...
---

# T-M5-02 — Fertility — BBT shift engine (3-over-6) and GET /fertility/bbt

## Why
Artboard `v19_TTC_BBT`: chart with coverline, fertile window band, shift detection and stats.

## Scope
1. `internal/fertility/bbt` (pure, clock-injected): per-cycle series by cycle day (cycle boundaries from the cycle
   engine's period history), coverline + 3-over-6 shift detection, phase, pre-ovulation average (first 6 readings),
   logged days vs cycle days so far, gaps, past shift days of previous cycles.
2. `GET /api/v1/fertility/bbt?range=1|3|6` (README shape) with the localized tip («دنبال جهش ۰٫۲ تا ۰٫۵ درجه
   باش…», mentioning past shift days when known).
3. Table-driven tests: clean biphasic, no shift, noisy with gaps, fewer than 6 readings, range 3/6 alignment.

## Acceptance
- Engine tests cover every branch; endpoint integration test green.
