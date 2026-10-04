---
id: L5-05
title: Business landing + multi-step join form + join-done page
milestone: L5
type: fullstack
status: todo
depends_on: [L5-01,L3-09,L3-01]
parallel_group: L5-C
touches: [resources/views/pages/directory/business.blade.php,resources/views/pages/directory/join.blade.php,resources/views/pages/directory/join-done.blade.php,app/Domain/Directory/Join,database/migrations,app/Http/Controllers/Directory/JoinController.php,app/Http/Requests/JoinRequest.php,resources/js/modules/stepper.js,tests/Feature/Directory/JoinTest.php]
skills: []
verify: composer verify && node tools/shot.mjs --design directory-business.html --route /directory/business && node tools/shot.mjs --design directory-join.html --route /directory/join && node tools/shot.mjs --design directory-join-done.html --route /directory/join/done
---

# L5-05 — Business landing + multi-step join form + join-done page

## Scope
- `/directory/business` (why/how/requirements/terms + FAQ group `directory-business`, `WebPage` + `FAQPage`).
- `/directory/join`: steps (info, location (address + optional lat/lng text — no map widget), images upload
  (through the media pipeline, max 10, client-side size check), age range, amenities, hours) — progressive: one
  long form without JS, stepper module with JS; server validates everything; `JoinRequest` stored with media
  attached as pending.
- `/directory/join/done` (noindex) matching design.

## Acceptance
- Full submit with images → JoinRequest + optimised media; three pages diff < 3%; tests green.
