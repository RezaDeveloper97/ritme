---
id: CB-DIR-01
title: Directory schema on bloom's city services: places, services, slots, bookings, reviews, applications
epic: DIR
type: backend
status: todo
depends_on: [B-N10-05, CB-CORE-05]
parallel_group: DIR-A
touches: [backend-go/db/migrations, backend/database/migrations, backend-go/db/queries/cityservices, backend-go/sqlc.yaml, docs/canvas-build/directory.md]
skills: [new-endpoint]
boards: [nbl_Dir_Place.dc.html, nbl_Dir_Book.dc.html, nbl_Dir_JoinForm.dc.html, nbl_Dir_JoinDocs.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && make schema-diff
---

# CB-DIR-01 — Directory schema on bloom's city services: places, services, slots, bookings, reviews, applications

## Why
bloom has admin listings + booking link/request (B-N10-05). The canvas adds map, rich place pages, slots, reviews and business applications (MVP: no online payment, admin-managed). Built ON TOP of the bloom/ queue: never re-implement what a `B-N*` dependency delivers — extend it.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Dir_Place.dc.html`
- `nbl_Dir_Book.dc.html`
- `nbl_Dir_JoinForm.dc.html`
- `nbl_Dir_JoinDocs.dc.html`

## Scope
1. Extend `cityservices`: place fields (district, lat/lng, age ranges, amenities, weekly hours, about, public photos, verified, status, booking mode online/request/phone, cancel policy, rules), services (duration, age range, capacity, price, packages), slots, bookings (child from bloom children, status requested/confirmed/rejected/cancelled/completed, code), reviews (completed bookings only, sub-scores), reports, business_applications (4-step draft, private docs, tracking code).
2. Catalog seeds dir_categories, dir_amenities, dir_age_ranges.
3. (CB-CORE-01) DECISIONS #13/#14: booking mode `online` (board «رزرو آنلاین با تقویم ریتمی») still creates a **request** confirmed by the Ritme team in admin — no auto-confirm, no payment fields; «پیام به مجموعه» has no messaging backend (hide or route to support).

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- schema-diff green.
- `verify` green.
