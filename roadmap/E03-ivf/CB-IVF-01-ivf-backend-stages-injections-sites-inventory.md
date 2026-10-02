---
id: CB-IVF-01
title: IVF backend: stages, injections, sites, inventory, scans, TWW
epic: IVF
type: backend
status: done
depends_on: [B-N2-03, B-N4-02]
parallel_group: IVF-A
touches: [backend-go/internal/ivf, backend-go/internal/http/routes_ivf.go, backend-go/db/queries/ivf, backend-go/api/openapi.yaml, backend-go/contract, backend-go/resources/translations, backend-go/db/migrations, backend/database/migrations, docs/go-migration/deviations.md]
skills: [new-endpoint]
boards: [nbl_IVF_Home.dc.html, nbl_IVF_Meds.dc.html, nbl_IVF_Scan.dc.html, nbl_IVF_TWW.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/ivf/... && make test-int PKG=./internal/ivf/... && golangci-lint run && make schema-diff
---

# CB-IVF-01 — IVF backend: stages, injections, sites, inventory, scans, TWW

## Why
bloom has only the IVF/IUI toggle (B-N2-03). Built ON TOP of the bloom/ queue: never re-implement what a `B-N*` dependency delivers — extend it.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_IVF_Home.dc.html`
- `nbl_IVF_Meds.dc.html`
- `nbl_IVF_Scan.dc.html`
- `nbl_IVF_TWW.dc.html`

## Scope
1. ivf_cycles (number, protocol, stage prep/stim/retrieval/transfer/tww/test + dates), ivf_meds (name, route, dose, time; trigger flag + exact time), dose logs, 8-site rotation with suggestion, inventory (units, days of supply, low flag), scans (per-ovary counts per bin <10/10–14/15–17/≥18, endometrium, E2), tww (transfer, beta date, mood, progesterone), outcome (positive → pregnancy setup, negative → loss path or cycle).
2. Injection/appointment reminders via care reminders; companion copy through bloom companion grants (B-N4-02).

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- OpenAPI entry + contract case for every new route; handler + service tests (unit + int).
- Schema change → goose migration + Laravel schema-only twin, `make schema-diff` green (backend-go/CLAUDE.md).
- New Go-only route group listed in docs/go-migration/deviations.md.
- `verify` green.
