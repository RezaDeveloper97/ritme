# Pwa

Web app manifest, install metadata and icons (L8-01); service worker versioning and the two-tier update policy (L8-02).

- `Manifest/WebManifest` — `/manifest.webmanifest` JSON + `InstallMetadata` for `<x-pwa.head>`, cache-aside in the
  `settings` namespace (key includes the `media` version). Source: `Settings\Data\PwaSettings`.
- `Manifest/IconSetResolver` — generated set from the admin logo (`icon_media_id`) or the defaults in `public/icons`.
- `Actions/GeneratePwaIcons` — builds the set from a raster logo (≥ 512 px) into `pwa/{id}-{version}/` on the media disk.
- `Support/PwaIconFiles` — file names, sizes, screenshots. Defaults are regenerated with `node tools/pwa-icons.mjs`.

Bindings live in `App\Providers\Domain\PwaServiceProvider` (none needed yet). See `docs/ARCHITECTURE.md`.
