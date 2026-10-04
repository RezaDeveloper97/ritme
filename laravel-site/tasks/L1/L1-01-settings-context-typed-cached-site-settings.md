---
id: L1-01
title: Settings context: typed, cached site settings
milestone: L1
type: backend
status: done
depends_on: [L0-03]
parallel_group: L1-A
touches: [app/Domain/Settings,database/migrations,database/seeders/SettingsSeeder.php,tests/Feature/Settings]
skills: []
verify: composer verify
---

# L1-01 — Settings context: typed, cached site settings

## Why
App-store links, contact info, socials, enamad, SEO defaults and Organization schema data appear on every page and
must be admin-editable without a deploy.

## Scope
- `settings` table (group, key, value JSON, unique group+key). Typed setting groups as readonly DTOs
  (`GeneralSettings`, `ContactSettings`, `SocialSettings`, `AppLinksSettings` (Bazaar, Myket, Google Play, App Store,
  PWA/web-app URL), `SeoDefaults` (title template `%s — ریتمی`, separator, default description, default OG media id,
  twitter handle, verification codes), `OrganizationSettings` (legal name, logo media id, founding date, sameAs,
  contact point), `LegalSettings` (enamad code/html allowed list, data-protection email), `PwaSettings`).
- `SettingsRepository` contract + Eloquent impl + `CachedSettingsRepository` (cache-aside `settings` ns, forever
  until bump); a view composer shares only the groups the layout needs.
- Seeder with the values/placeholders found in `docs/AUDIT.md`.
- Feature tests: read, update bumps cache, unknown key throws.

## Out of scope
- Filament settings pages (L1-08 creates the shell, L7-06 completes SEO/Org settings).

## Acceptance
- One query (or zero when cached) per request for settings; tests green.
