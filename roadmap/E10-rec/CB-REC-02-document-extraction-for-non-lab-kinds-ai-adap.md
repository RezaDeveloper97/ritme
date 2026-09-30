---
id: CB-REC-02
title: Document extraction for non-lab kinds (AI adapter, Plus) + pregnancy dating hook
epic: REC
type: backend
status: todo
depends_on: [CB-REC-01, B-N6-05, B-N6-06, B-N2-06]
parallel_group: REC-B
touches: [backend-go/internal/healthrecord/extract, backend-go/internal/healthrecord, backend-go/internal/pregnancy, backend-go/api/openapi.yaml]
skills: [new-endpoint]
boards: [nbl_Rec_Doc.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/healthrecord/... && make test-int PKG=./internal/healthrecord/... && golangci-lint run
---

# CB-REC-02 — Document extraction for non-lab kinds (AI adapter, Plus) + pregnancy dating hook

## Why
'اطلاعات خوانده‌شده از سند — اگر اشتباه است ویرایش کن'. AI features are Plus (DECISIONS).

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Rec_Doc.dc.html`

## Scope
1. Async extraction through bloom's AI adapter (fake by default) with per-kind JSON schema (date, centre, doctor, GA, EDD, findings); `needs_review` until the user confirms; Plus-gated via entitlements (B-N2-06), free users fill fields manually.
2. Confirmed imaging doc with GA/EDD in pregnancy mode → offer dating update (explicit confirm, never silent). No file content in logs.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Fake-provider tests per kind; GA update needs confirm; free user path tested.
- `verify` green.
