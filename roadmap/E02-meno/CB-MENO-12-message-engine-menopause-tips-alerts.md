---
id: CB-MENO-12
title: Message engine: menopause tips & alerts
epic: MENO
type: backend
status: todo
depends_on: [CB-MENO-02]
parallel_group: MENO-C
touches: [backend-go/internal/messages, backend-go/db/migrations, backend/database/migrations, backend-go/resources/translations]
skills: [new-endpoint]
boards: [Main.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/messages/... && make test-int PKG=./internal/messages/... && golangci-lint run && make schema-diff
---

# CB-MENO-12 — Message engine: menopause tips & alerts

## Why
Today surfaces menopause advice like other modes (if B-N9-05 messages v2 is done, add rules there instead).

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `Main.dc.html`

## Scope
1. Rules: bleeding after menopause (high), checkup due/overdue, score +4 worse, HRT review near; stage tips from admin content.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Rules unit-tested; texts admin-editable.
- `verify` green.
