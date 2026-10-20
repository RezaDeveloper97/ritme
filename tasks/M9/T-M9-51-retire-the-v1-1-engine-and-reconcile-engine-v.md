---
id: T-M9-51
title: Retire the v1.1 engine and reconcile engine-v6 deviations
milestone: M9
type: backend
status: todo
depends_on: [T-M9-50]
parallel_group: M9-R
touches: [backend-go/internal/cycle, backend-go/internal/clinicalconfig, backend-go/contract, backend-go/api/openapi.yaml, backend-go/internal/platform/config, docs/go-migration/deviations.md, docs/engine-v6/README.md]
skills: []
verify: cd backend-go && go vet ./... && go test ./... && make test-int PKG=./internal/cycle/... && make contract ROUTES=all
---

# T-M9-51 — Retire the v1.1 engine and reconcile engine-v6 deviations
**Model:** opus; the main session reviews the deletions and the deviation rows.

## Why
Three engines and two caches must not live on. Once beta has run v6 without incident, v1.1 code, its PHP-generated
enum coupling, the content-hash cache and the legacy payload keys go (X-01, X-04, X-05, D-139).

## Scope
1. Delete `internal/cycle/resolver`, the legacy-key builders in `internal/cycle/view`, the legacy engine paths in
   `internal/cycle/legacy` that only served v1, `internal/cycle/cache` (v1 cache), `cycle/model` adapters no longer
   used, and v1 branches behind `CYCLE_ENGINE` (the flag is removed; v6 is the only engine).
2. Clinical config: retire `engine.fertile_window.after` (registry field removed; stored versions with the field still
   parse); source strings updated.
3. Contract: Laravel cycle goldens are no longer replayed (keep the files archived under `contract/golden/_laravel-v11/`
   or delete them — ask the user); Go-native v6 goldens are the reference; allow-lists cleaned.
4. Legacy payload keys (`phase`, `data_status`, `predictions`, value layers, `data_quality_details`, `daily_card`,
   per-day `calculation`) removed from the Go output **only if** the frontend no longer reads them from a Go backend
   (grep + the T-M9-40 fallback paths); otherwise keep them and record why. OpenAPI updated.
5. `docs/go-migration/deviations.md`: reconcile D-120…D-139 (status → shipped on stage/beta with dates; D-30, D-55
   point to successors) — the release task is the only M9 task allowed to edit this file besides T-M9-01.
6. README §1 "Where we start" moved to a history note; §7 updated.

## Tests
- Full `go test ./...`, integration for cycle, `make contract ROUTES=all` with the new allow-lists.
- grep test: no import of the deleted packages remains.

## Acceptance
- Verify passes; deviations reconciled; PROGRESS.md cites D-139.
