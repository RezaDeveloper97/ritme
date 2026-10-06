---
id: CB-MENO-03
title: Menopause API: treatment + report section
epic: MENO
type: backend
status: done
depends_on: [CB-MENO-01, B-N6-04]
parallel_group: MENO-B
touches: [backend-go/internal/menopause/treatment, backend-go/internal/menopause/report, backend-go/internal/healthrecord, backend-go/internal/http/routes_menopause.go, backend-go/db/queries/menopause, backend-go/api/openapi.yaml, backend-go/contract]
skills: [new-endpoint]
boards: [nbl_Meno_Treatment.dc.html, nbl_Meno_Report.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/menopause/... && make test-int PKG=./internal/menopause/... && golangci-lint run
---

# CB-MENO-03 — Menopause API: treatment + report section

## Why
Treatment & care, and the doctor report — as a menopause section of bloom's report builder (B-N6-04), not a second builder.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Meno_Treatment.dc.html`
- `nbl_Meno_Report.dc.html`

## Scope
1. Treatment CRUD, intake log (weekly dots), side effects, lifestyle goal progress, review date; reminders via care-reminders (M3).
2. Report section data for 1/3/6 months: stage, score first→last, flash avg/day, night-sweat nights/week, sleep avg, bleeding events, BP avg (vitals B-N6-01 if present), top symptoms % of days, adherence %, side effects, supplements, questions text.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- OpenAPI entry + contract case for every new route; handler + service tests (unit + int).
- `verify` green.
