---
id: B-N6-04
title: Doctor report builder, PDF and 7-day share link
milestone: N6
type: fullstack
status: done
depends_on: [B-N6-03]
parallel_group: N6-D
touches: [backend-go/internal/healthrecord,backend-go/internal/sharelinks,backend-go/api,frontend/src/screens/record-export,frontend/src/shared/lib/pdf]
skills: [new-endpoint,verify-all,security-review]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N6-04 — Doctor report builder, PDF and 7-day share link

## Why
Report for the doctor.

## Design
- `docs/design/night-bloom/c-health-record/nbl_Record_Export.dc.html` (+ `nbd_Record_Export`)
- `docs/design/night-bloom/c-health-record/nbl_Record_Preview.dc.html` (+ `nbd_Record_Preview`)

## Scope
- Range + section toggles + patient question, preview (2 pages), PDF built on device (existing pdf lib) — download; share link: encrypted upload, tokenised public URL valid 7 days, revocable, listed in Privacy; Plus-gated share.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- Expired/revoked link returns 410
- `verify` green
