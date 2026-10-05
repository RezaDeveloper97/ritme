# Seo / Indexing (L7-04)

Admin indexing controls, stored as flat keys of the `seo` settings group (`Settings\Data\IndexingSettings`, exposed as
`SeoDefaults::$indexing`) and edited on `/admin/seo/indexing` (`App\Filament\Pages\Seo\IndexingSettingsPage`, SEO
manager + super-admin; head code super-admin only; every change in the activity log `seo`).

- `SettingsRobotsRules` (bound to `Sitemap\RobotsRules`): admin robots.txt or `DefaultRobotsRules`; validated by
  `RobotsTxtValidator`. Production only — `RobotsTxt` keeps `Disallow: /` elsewhere.
- `IndexingRules`: per-type robots default (`IndexingType`, keyed by sitemap provider key) used by `SeoManager`
  (controller > seo_meta > type default > index,follow) and the sitemap exclude / priority / changefreq used by `Sitemaps`.
- `Observers/IndexingSettingObserver`: any `seo` setting write bumps `sitemap` (robots + sitemap documents).
- `HeadCode`: allow-listed same-origin `<meta>`/`<link>` only, rebuilt on render — never scripts (strict CSP, no externals).
- IndexNow (`IndexNow/`, `Jobs/SubmitToIndexNow`, `Observers/IndexNowObserver` on Post/Product/Place): off by default,
  production only, queued server-side call; key file `/{key}.txt` (`IndexNowKeyController`, route in `SeoServiceProvider`).
- `Actions/RegenerateSitemaps` (+ `Jobs/PingSitemaps`, production only), `ResetRobotsTxt`, `RegenerateIndexNowKey`,
  `SaveIndexingSettings`.
