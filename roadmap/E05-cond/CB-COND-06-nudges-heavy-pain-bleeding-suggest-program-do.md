---
id: CB-COND-06
title: Nudges: heavy pain/bleeding → suggest program & doctor
epic: COND
type: backend
status: done
depends_on: [CB-COND-01]
parallel_group: COND-B
touches: [backend-go/internal/messages, backend-go/resources/translations]
skills: [new-endpoint]
boards: [IA_Map.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/messages/... && make test-int PKG=./internal/messages/... && golangci-lint run
---

# CB-COND-06 — Nudges: heavy pain/bleeding → suggest program & doctor

## Why
IA_Map cross-link 'ثبت درد زیاد → پیشنهاد دفترچه درد و پزشک'.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `IA_Map.dc.html`

## Scope
1. Rules: pain ≥7 on 2+ days in a cycle (not enrolled) → pain diary; heavy flow 2+ days → PBAC; links to program (and doctors when N7 exists); admin-editable texts.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Rules unit-tested.
- `verify` green.
