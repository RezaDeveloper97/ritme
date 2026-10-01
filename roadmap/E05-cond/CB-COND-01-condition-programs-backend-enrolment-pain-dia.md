---
id: CB-COND-01
title: Condition programs backend: enrolment, pain diary, PMDD, PBAC
epic: COND
type: backend
status: done
depends_on: [CB-CORE-03, B-N3-01]
parallel_group: COND-A
touches: [backend-go/internal/conditions, backend-go/internal/http/routes_conditions.go, backend-go/db/queries/conditions, backend-go/api/openapi.yaml, backend-go/contract, backend-go/resources/translations, backend-go/db/migrations, backend/database/migrations, docs/go-migration/deviations.md]
skills: [new-endpoint]
boards: [nbl_Cond_Hub.dc.html, nbl_Cond_Endo.dc.html, nbl_Cond_PMDD.dc.html, nbl_Cond_Bleed.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/conditions/... && make test-int PKG=./internal/conditions/... && golangci-lint run && make schema-diff
---

# CB-COND-01 — Condition programs backend: enrolment, pain diary, PMDD, PBAC

## Why
bloom lists programs as admin content in the services hub (B-N7-01); the canvas makes them real trackers. Built ON TOP of the bloom/ queue: never re-implement what a `B-N*` dependency delivers — extend it.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Cond_Hub.dc.html`
- `nbl_Cond_Endo.dc.html`
- `nbl_Cond_PMDD.dc.html`
- `nbl_Cond_Bleed.dc.html`

## Scope
1. condition_enrolments (endo / pmdd / heavy_bleeding / pcos).
2. Pain diary reuses taxonomy pain (locations, 1–10, relief — B-N3-01) plus program fields: types[], associated[], missed work/school, analgesic + effect.
3. pmdd_entries (6 items 1–6) + chart data for the last 2 cycles + 'needs 2 complete cycles' rule.
4. pbac_entries (pads per level ×1/5/20, clots, flooding) → day/period score, ≥100 alert text.
5. PCOS: enrolment only, uses existing logs (DECISIONS minimal). Catalog: pain_types, pain_associated, pmdd_items.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- OpenAPI entry + contract case for every new route; handler + service tests (unit + int).
- Schema change → goose migration + Laravel schema-only twin, `make schema-diff` green (backend-go/CLAUDE.md).
- New Go-only route group listed in docs/go-migration/deviations.md.
- `verify` green.
