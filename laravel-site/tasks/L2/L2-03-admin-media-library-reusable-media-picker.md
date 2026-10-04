---
id: L2-03
title: Admin media library + reusable media picker
milestone: L2
type: admin
status: todo
depends_on: [L2-01,L1-08]
parallel_group: L2-C
touches: [app/Filament/Resources/Media,app/Filament/Forms/Components,tests/Feature/Admin/MediaTest.php]
skills: []
verify: composer verify
---

# L2-03 — Admin media library + reusable media picker

## Why
Editors need one place to upload, describe (alt text = SEO) and reuse images.

## Scope
- Filament `MediaResource`: grid view with thumbs, multi-upload, alt/title/caption edit (alt required; warning badge
  "بدون متن جایگزین" in the grid), focal-point picker (click on image), file size / dimensions / variants list with
  savings %, "regenerate variants" action, "find usages", bulk delete of unused only.
- `MediaPicker` form field (select existing or upload new) used by every later resource; mobile override image
  field variant for art direction.
- Permissions: editors+.

## Acceptance
- Upload in admin → variants created (sync or via queue worker) → visible in grid with savings; tests green.
