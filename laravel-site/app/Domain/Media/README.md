# Media

Upload/optimise pipeline, responsive variants and the media library behind <x-picture>.

- `Actions/StoreMedia` (upload: sniff + validate + dedupe + auto-orient/strip + capped original + LQIP/dominant
  colour, then queues `Jobs/OptimizeMedia`) and `Actions/GenerateMediaVariants` (presets from `config/media.php`).
- `Contracts/MediaRepository` → `Repositories/CachedMediaRepository` (`media` ns) → `EloquentMediaRepository`;
  views get `Data/MediaData` (URLs resolved, missing variants fall back to the original).
- `Observers/MediaObserver` bumps `media`, `seo`, `pages` and deletes the files of deleted rows.
- `Support/MediaOgImageResolver` implements the Seo `OgImageResolver` (og variant).
- `php artisan media:regenerate {--id=*} {--preset=*}`.

Bindings live in `App\Providers\Domain\MediaServiceProvider`. See `docs/ARCHITECTURE.md`.
