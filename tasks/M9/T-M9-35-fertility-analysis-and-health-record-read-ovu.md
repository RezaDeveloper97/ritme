---
id: T-M9-35
title: Fertility, analysis and health record read ovulation and regularity from the engine
milestone: M9
type: backend
status: done
depends_on: [T-M9-32]
parallel_group: M9-C
touches: [backend-go/internal/fertility, backend-go/internal/http/routes_fertility.go, backend-go/internal/http/routes_analysis.go, backend-go/internal/http/routes_analysis_ttc.go, backend-go/internal/http/routes_healthrecord.go, backend-go/internal/http/routes_sharelinks.go, backend-go/internal/http/routes_recordsharing.go, backend-go/internal/http/routes_assistant.go, backend-go/internal/analysis, backend-go/internal/healthrecord, backend-go/internal/assistant/profile.go, backend-go/api/openapi.yaml, backend-go/contract/allowlist/fertility.yaml, backend-go/contract/allowlist/analysis.yaml, backend-go/contract/allowlist/analysis-ttc.yaml]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/fertility/... ./internal/analysis/... ./internal/healthrecord/... ./internal/assistant/... && make test-int PKG="./internal/fertility/... ./internal/analysis/..." && make contract ROUTES=fertility,analysis,analysis-ttc,healthrecord
---

# T-M9-35 — Fertility, analysis and health record read ovulation and regularity from the engine
**Model:** opus (health logic).

## Why
Fertility insights and TTC analysis keep their own BBT > LH > estimate logic (no conflict rule), `bbt.go` uses a
different rise rule, `/fertility/insights` runs the resolver ~185–380 times per request, and the analysis tab has a
second FIGO regularity with a one-outlier exclusion, so the tab and Today can disagree (gap 02 §5, gap 07 §4.5).
DECISIONS.md E-09, A-01, D-128, D-129.

## Scope
1. Fertility (v6/shadow mode): per-cycle ovulation, source and fertile window come from the `State` (BBT via
   `engine/bbt`, LH, conflict); the BBT chart uses `engine/bbt` for its coverline/shift; no per-day resolve loops (one
   `Range` per cycle or the prepared per-cycle data). `internal/fertility/bbt` delegates to `engine/bbt` in v6 mode and
   keeps its old rule in v1 mode (Laravel goldens).
2. Analysis: regularity verdict = engine `cycle_regularity` (FIGO band shown as reference only, A-01); remove
   `markOutlier` (E-09) in v6 mode; TTC ovulation source per cycle from the engine; `typical()` statistics may stay
   window-based but never contradict the engine verdict; enum `not_enough_data` → `insufficient_data` in v6 mode
   (OpenAPI documents both).
3. Health record and assistant profile: `last_period_start` = latest actual start (no declared anchors), cycle section
   from the engine (effective lengths, regularity).
4. Shared state provider; `LocalToday`; ETag on fertility today/insights/bbt.

## Tests
- BBT strict fixture (third reading below +0.20) rejected; 9 readings unconfirmed; LH/BBT conflict shown with BBT.
- `/fertility/insights` resolves ≤ 1 range per cycle (bench/counter); ≤ 4 queries.
- Agreement: analysis regularity == `cycle_view.metrics.cycle_regularity` for 8 personas.
- Contract: v1 unchanged; v6 Go-native goldens for fertility/analysis personas.

## Budgets
- `/fertility/insights` warm ≤ 25 ms p95, ≤ 4 queries.

## Out of scope
- Analysis caching and new analysis methods (T-M10-26, T-M10-28).

## Acceptance
- Tests green; PROGRESS.md cites D-128 and D-129.
