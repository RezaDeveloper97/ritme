---
id: CB-LOSS-01
title: Loss path backend: event, stop pregnancy content, follow-up, mood, next step
epic: LOSS
type: backend
status: todo
depends_on: [B-N2-03, B-N4-02, CB-CORE-03]
parallel_group: LOSS-A
touches: [backend-go/internal/loss, backend-go/internal/http/routes_loss.go, backend-go/db/queries/loss, backend-go/api/openapi.yaml, backend-go/contract, backend-go/resources/translations, backend-go/db/migrations, backend/database/migrations, docs/go-migration/deviations.md, backend-go/internal/pregnancy, backend-go/internal/messages]
skills: [new-endpoint]
boards: [nbl_Loss_Start.dc.html, nbl_Loss_Care.dc.html, nbl_Loss_Next.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/loss/... && make test-int PKG=./internal/loss/... && golangci-lint run && make schema-diff
---

# CB-LOSS-01 — Loss path backend: event, stop pregnancy content, follow-up, mood, next step

## Why
bloom only adds a calm exit option (B-N2-03); the canvas has a full 3-step path. Built ON TOP of the bloom/ queue: never re-implement what a `B-N*` dependency delivers — extend it.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Loss_Start.dc.html`
- `nbl_Loss_Care.dc.html`
- `nbl_Loss_Next.dc.html`

## Scope
1. pregnancy_losses (type, approx date); on create stop pregnancy content/reminders/messages; optional companion notice (bloom grant, one line only).
2. Follow-up via care reminders: daily bleeding until stopped, beta until negative, visit ~2 weeks; mood logs; encrypted private note (never shared).
3. Next step: cycle only / ttc again / nothing → life mode; recurrent-loss hint when ≥2.
4. Catalog seeds loss_warning_signs, loss_hotlines (1480, 123, 115) `[needs review]`.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- OpenAPI entry + contract case for every new route; handler + service tests (unit + int).
- Schema change → goose migration + Laravel schema-only twin, `make schema-diff` green (backend-go/CLAUDE.md).
- New Go-only route group listed in docs/go-migration/deviations.md.
- `verify` green.
