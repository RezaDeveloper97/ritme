---
id: L5-04
title: Booking request flow + booked page
milestone: L5
type: fullstack
status: done
depends_on: [L5-03]
parallel_group: L5-C
touches: [app/Domain/Directory/Booking,database/migrations,app/Http/Controllers/Directory/BookingController.php,app/Http/Requests/BookingRequest.php,resources/views/pages/directory/booked.blade.php,app/Notifications,tests/Feature/Directory/BookingTest.php]
skills: []
verify: composer verify && node tools/shot.mjs --design directory-booked.html --route /directory/booked/DEMO
---

# L5-04 — Booking request flow + booked page

## Scope
- Booking form on place page (service, preferred date (Jalali picker module, no-JS text input fallback), time
  window, name, phone, notes) → `BookingRequest` (code, status new|confirmed|cancelled|done) via
  `CreateBookingRequest` action in a transaction; PRG → `/directory/booked/{code}` (noindex, signed or unguessable
  code) matching `directory-booked.html`.
- Notifications: admin + optional place contact (mail/SMS via `SmsSender` contract with a `log` driver default).
- Rate limit per IP/phone; phone normalisation (Persian digits → Latin, +98).

## Acceptance
- Submit → booked page diff < 3%; spam rules tested; tests green.
