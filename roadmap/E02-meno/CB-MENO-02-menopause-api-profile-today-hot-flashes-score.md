---
id: CB-MENO-02
title: Menopause API: profile, today, hot flashes, score, patterns
epic: MENO
type: backend
status: todo
depends_on: [CB-MENO-01, B-N3-07]
parallel_group: MENO-B
touches: [backend-go/internal/menopause, backend-go/internal/http/routes_menopause.go, backend-go/db/queries/menopause, backend-go/api/openapi.yaml, backend-go/contract, backend-go/resources/translations]
skills: [new-endpoint]
boards: [nbl_Meno_Stage.dc.html, nbl_Meno_Home.dc.html, nbl_Meno_HotFlash.dc.html, nbl_Meno_Score.dc.html, Main.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/menopause/... && make test-int PKG=./internal/menopause/... && golangci-lint run
---

# CB-MENO-02 — Menopause API: profile, today, hot flashes, score, patterns

## Why
Endpoints behind home, hot-flash timer and score.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Meno_Stage.dc.html`
- `nbl_Meno_Home.dc.html`
- `nbl_Meno_HotFlash.dc.html`
- `nbl_Meno_Score.dc.html`
- `Main.dc.html`

## Scope
1. `/api/v1/menopause`: profile GET/PUT (months without period + stage label), today (flash count, night sweats, last sleep, score + 6-month trend, upcoming checkups, treatment adherence), hot flash start/stop/list, monthly score submit/history with domains + delta.
2. Patterns through bloom's analysis engine (B-N3-07) where it fits (trigger ↔ flash frequency, night sweat ↔ next-day fatigue), min-sample thresholds, 'not a diagnosis' flag.
3. Bleeding item logged while stage is meno/post → alert flag in today.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- OpenAPI entry + contract case for every new route; handler + service tests (unit + int).
- `verify` green.
