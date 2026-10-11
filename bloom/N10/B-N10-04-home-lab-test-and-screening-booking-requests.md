---
id: B-N10-04
title: Home lab test and screening booking requests
milestone: N10
type: fullstack
status: done
depends_on: [B-N10-01, B-N6-06]
parallel_group: N10-D
touches: [backend-go/internal/partners,backend-go/api,admin-web/src,frontend/src/screens/home-lab]
skills: [new-endpoint,verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N10-04 — Home lab test and screening booking requests

## Why
Home suggestions «آزمایش در منزل» / «چکاپ سالانه — رزرو».

## Scope
- `internal/partners`: admin catalog of orderable services (home blood draw, annual checkup / Pap) and an admin request queue; `LabPartner` adapter (`Submit`/`Status`/`FetchResult`) with `fake` default (admin-advanced statuses, canned result PDF).
- Request form (service, name, phone, city/address, preferred day + time window, free-text note with a «don't write medical details» warning) → `POST/GET /api/v1/partners/lab-requests` + cancel; statuses requested → accepted → sampled → result_ready / rejected / cancelled. Home «پیشنهاد» cards shown only when a service is active and are never driven by health logs.
- Partner payload = allow-listed form fields only, explicit consent line naming the partner; results are copied into labs (B-N6-06) only with the user's consent; address purged after the retention window.
- No dedicated artboards exist: build from existing components, light + dark.
- Full detail: `docs/night-bloom/extras.md` §3.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
