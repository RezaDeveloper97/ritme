---
id: CB-CONTRA-01
title: Contraception backend: method, pill pack, streak, long-acting reminders
epic: CONTRA
type: backend
status: todo
depends_on: [B-N2-03, CB-CORE-03]
parallel_group: CONTRA-A
touches: [backend-go/internal/contraception, backend-go/internal/http/routes_contraception.go, backend-go/db/queries/contraception, backend-go/api/openapi.yaml, backend-go/contract, backend-go/resources/translations, backend-go/db/migrations, backend/database/migrations, docs/go-migration/deviations.md]
skills: [new-endpoint]
boards: [nbl_Contra_Setup.dc.html, nbl_Contra_Pill.dc.html, nbl_Contra_Missed.dc.html, nbl_Contra_Other.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/contraception/... && make test-int PKG=./internal/contraception/... && golangci-lint run && make schema-diff
---

# CB-CONTRA-01 — Contraception backend: method, pill pack, streak, long-acting reminders

## Why
bloom's mode switcher has the 'track contraception' switch (B-N2-03) but nothing behind it. Built ON TOP of the bloom/ queue: never re-implement what a `B-N*` dependency delivers — extend it.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Contra_Setup.dc.html`
- `nbl_Contra_Pill.dc.html`
- `nbl_Contra_Missed.dc.html`
- `nbl_Contra_Other.dc.html`

## Scope
1. contraception_methods (combined/progestin pill, copper/hormonal IUD, injection, implant, condom, other; pack type 21+7/28/24+4, pack start, reminder time, IUD lifetime, insert/injection/implant dates).
2. Pill logs, pack day, placebo days, streak, next pack, refill reminder 5 days before; IUD monthly string check + replacement + 6-week check; injection every 12 weeks; implant date — all via care reminders.
3. Catalog missed_pill_rules `[needs review]`.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- OpenAPI entry + contract case for every new route; handler + service tests (unit + int).
- Schema change → goose migration + Laravel schema-only twin, `make schema-diff` green (backend-go/CLAUDE.md).
- New Go-only route group listed in docs/go-migration/deviations.md.
- `verify` green.
