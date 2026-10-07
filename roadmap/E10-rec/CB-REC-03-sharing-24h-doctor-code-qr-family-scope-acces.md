---
id: CB-REC-03
title: Sharing: 24h doctor code + QR, family scope, access log, emergency card
epic: REC
type: backend
status: done
depends_on: [CB-REC-01, B-N6-04, B-N4-02]
parallel_group: REC-B
touches: [backend-go/internal/sharelinks, backend-go/internal/healthrecord, backend-go/internal/http/routes_healthrecord.go, backend-go/db/migrations, backend/database/migrations, backend-go/api/openapi.yaml]
skills: [new-endpoint]
boards: [nbl_Rec_Share.dc.html, nbl_Rec_Emergency.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/healthrecord/... && make test-int PKG=./internal/healthrecord/... && golangci-lint run && make schema-diff
---

# CB-REC-03 — Sharing: 24h doctor code + QR, family scope, access log, emergency card

## Why
Extends bloom's 7-day share links (B-N6-04) with the canvas' doctor code and access history.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Rec_Share.dc.html`
- `nbl_Rec_Emergency.dc.html`

## Scope
1. Share-link option: 24h summary with short code + QR; access log per view (who/when, link id); revoke.
2. Family scope through bloom companion grants: meds & appointments only, never documents.
3. Emergency card data (name, blood group, allergies, conditions, permanent meds, optional pregnancy status, masked insurance, emergency contact) + show-on-lock-screen flag.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Expired/revoked → 404; access log written.
- OpenAPI entry + contract case for every new route; handler + service tests (unit + int).
- `verify` green.
