---
id: B-N10-05
title: City services — classes, pools, playhouses and bookings
milestone: N10
type: fullstack
status: done
depends_on: [B-N10-01]
parallel_group: N10-E
touches: [backend-go/internal/cityservices,backend-go/api,admin-web/src,frontend/src/screens/city-services]
skills: [new-endpoint,verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N10-05 — City services — classes, pools, playhouses and bookings

## Why
«برای مادر و کودک» + «رزروها» in Me.

## Design
- `docs/design/night-bloom/d-doctor-assistant/nbl_v17_Main.dc.html` (+ `nbd_v17_Main`)

## Scope
- `internal/cityservices`: admin-managed cities, categories (class / pool / playhouse) and listings (name, city, district, address, phone, description, price text, photos, hours, `booking_mode` link | request | phone, booking URL); list with city + category filter, detail, booking via link-out or a request (name, phone, preferred date, optional child age range, note) landing in an admin queue with user-visible status; thin «رزروها» list over the same table (CB-DIR-09 later supersedes it at `/profile/bookings`).
- No map, no online payment, no slots/reviews (CB-DIR). `BookingNotifier` adapter (`fake` = log). Nothing health-related (child medical data, cycle/pregnancy state) is sent to venues.
- Full detail: `docs/night-bloom/extras.md` §4.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
