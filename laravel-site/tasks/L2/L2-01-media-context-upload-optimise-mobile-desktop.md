---
id: L2-01
title: Media context: upload, optimise, mobile + desktop variants
milestone: L2
type: backend
status: done
depends_on: [L0-03,L1-01]
parallel_group: L2-A
touches: [app/Domain/Media,config/media.php,config/filesystems.php,database/migrations,app/Console/Commands/MediaRegenerate.php,tests/Feature/Media,tests/Unit/Media]
skills: []
verify: composer verify
---

# L2-01 — Media context: upload, optimise, mobile + desktop variants

## Why
The user requires every admin-uploaded image to be optimised with phone and desktop outputs. Images are usually the
largest GTmetrix penalty.

## Scope
- `intervention/image` v3, driver auto-detect (Imagick if loaded, else GD — cPanel usually has GD).
- `media` table: disk, directory, filename, mime, size, width, height, alt, title, caption, focal_x/y, dominant_color,
  lqip (tiny base64 WebP ≤ 600 bytes), variants JSON `{name:{format:{w,h,path,size}}}`, uploaded_by, hash (dedupe).
- `config/media.php` presets: `mobile` [480, 768], `desktop` [1280, 1920], `thumb` 320 (crop focal), `og` 1200×630
  (crop focal), `square` 600 (products); formats `avif` (only if `imageavif` + driver support), `webp`, original
  fallback (`jpg` for photos, `png` when alpha); qualities per format; **never upscale** (skip widths > original).
- Pipeline: validate (mime sniffing, max 15 MB, max 8000 px, reject SVG unless sanitised with
  `enshrined/svg-sanitize`), auto-orient by EXIF then strip metadata, compute dominant color + LQIP, write original
  (capped at 2560 px) + variants. `OptimizeMedia` job on the `database` queue; `QUEUE_CONNECTION=sync` works too;
  variants missing → original served (no broken images).
- Storage: `public` disk root `public_path('media')` by default (no `storage:link` needed on cPanel), URL
  `/media/...`; switchable by env.
- `media:regenerate {--id=} {--preset=}` command; deleting media deletes all files; `MediaObserver` bumps `media` ns.
- Tests with fixture JPEG/PNG (with EXIF rotation) / transparent PNG / animated GIF (kept as-is, poster variant),
  asserting dimensions, formats, metadata stripped, no upscale.

## Out of scope
- `<x-picture>` (L2-02), admin UI (L2-03).

## Acceptance
- Uploading a 4000×3000 JPEG yields avif/webp/jpg at 480/768/1280/1920 + thumb + og, all smaller than source;
  tests green.
