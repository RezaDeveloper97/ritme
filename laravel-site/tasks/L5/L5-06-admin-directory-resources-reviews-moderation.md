---
id: L5-06
title: Admin: directory resources, reviews moderation, bookings, join requests
milestone: L5
type: admin
status: todo
depends_on: [L5-04,L5-05,L2-03,L4-05]
parallel_group: L5-D
touches: [app/Filament/Resources/Directory,tests/Feature/Admin/DirectoryAdminTest.php]
skills: []
verify: composer verify
---

# L5-06 — Admin: directory resources, reviews moderation, bookings, join requests

## Scope
- Places (tabs: info, location, services & prices repeater, opening hours editor, gallery via MediaPicker, SEO via
  `SeoFields`), categories (schema type select), cities/districts, amenities (icon picker from sprite).
- City × category landing texts editor (title/description/h1/intro per combo).
- Reviews moderation queue (approve/reject, bulk), bookings board (status flow, filters, CSV), join requests
  (review → "convert to place draft" action copying data + media).
- `directory-manager` role policies.

## Acceptance
- Convert a join request into a published place visible on `/directory`; tests green.
