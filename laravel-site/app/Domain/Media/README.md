# Media

Upload/optimise pipeline, responsive variants and the media library behind <x-picture>.

- Variant files are versioned: `{stem}-{variant}.{hash10}.{ext}` (content hash), because `/media` is cached
  `immutable` (L1-07). A regenerate that changes pixels (focal move) produces new URLs; the new set is written
  before stale files are deleted. URLs always come from the stored `path`, so `MediaData`, `<x-picture>` and the OG
  resolver need no special handling; pre-versioning rows keep their stored paths until regenerated.
- `Actions/StoreMedia` (upload: sniff + validate + dedupe + auto-orient/strip + capped original + LQIP/dominant
  colour, then queues `Jobs/OptimizeMedia`) and `Actions/GenerateMediaVariants` (presets from `config/media.php`).
- `Contracts/MediaRepository` → `Repositories/CachedMediaRepository` (`media` ns) → `EloquentMediaRepository`;
  views get `Data/MediaData` (URLs resolved, missing variants fall back to the original).
- `Observers/MediaObserver` bumps `media`, `seo`, `pages` and deletes the files of deleted rows.
- `Support/MediaOgImageResolver` implements the Seo `OgImageResolver` (og variant).
- `php artisan media:regenerate {--id=*} {--preset=*}`.

Bindings live in `App\Providers\Domain\MediaServiceProvider`. See `docs/ARCHITECTURE.md`.
