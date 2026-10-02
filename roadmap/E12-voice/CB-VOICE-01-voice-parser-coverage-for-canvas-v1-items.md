---
id: CB-VOICE-01
title: Voice parser coverage for canvas-v1 items
epic: VOICE
type: backend
status: done
depends_on: [B-N3-05, CB-MENO-02, CB-COND-01, CB-CONTRA-01]
parallel_group: VOICE-A
touches: [backend-go/internal/voicelog, backend-go/api/openapi.yaml]
skills: [new-endpoint]
boards: [nbl_Voice_Review.dc.html, nbl_Voice_Saved.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/voicelog/... && make test-int PKG=./internal/voicelog/... && golangci-lint run
---

# CB-VOICE-01 — Voice parser coverage for canvas-v1 items

## Why
bloom builds voice logging (B-N3-05, Plus). Menopause, programs and contraception items must be understood and saved too.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Voice_Review.dc.html`
- `nbl_Voice_Saved.dc.html`

## Scope
1. Extend the parser schema + fixtures: menopause symptoms/hot flashes, pain-diary fields, pill taken, pelvic leak; ambiguity options (e.g. mood); commit writes to the new tables through their services.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Fixture tests per new item kind.
- `verify` green.
