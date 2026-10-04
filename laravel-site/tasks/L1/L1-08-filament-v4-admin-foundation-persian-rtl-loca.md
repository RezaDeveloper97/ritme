---
id: L1-08
title: Filament v4 admin foundation: Persian RTL, local fonts, roles, activity log
milestone: L1
type: admin
status: todo
depends_on: [L1-01]
parallel_group: L1-F
touches: [app/Filament,app/Providers/Filament,app/Models/User.php,database/migrations,database/seeders,resources/css/filament,config/filament.php,config/permission.php,tests/Feature/Admin]
skills: []
verify: composer verify
---

# L1-08 — Filament v4 admin foundation: Persian RTL, local fonts, roles, activity log

## Why
Every later domain task adds its admin resources into one consistent, secure panel.

## Scope
- Install Filament v4 (latest stable for Laravel 12); panel `admin` at `config('admin.path', 'admin')`; locale `fa`,
  RTL; **custom Vite theme** with self-hosted Vazirmatn (disable Filament's Bunny Fonts provider — no external font
  requests; verify in network log), brand colors from tokens, logo from settings.
- Users: `is_active`, `last_login_at`; spatie/laravel-permission roles `super-admin`, `editor`, `seo-manager`,
  `shop-manager`, `directory-manager`, `support` with policies per resource (deny by default). Filament MFA (app
  TOTP) available, required for super-admin.
- Login throttling, session timeout, `php artisan admin:create` command (interactive, no default password seeded).
- spatie/laravel-activitylog for all admin mutations; read-only "Activity" resource.
- Settings pages (Filament Pages) for General/Contact/Social/App links/Legal using L1-01 DTOs (SEO + Org settings come
  in L7-06). Saving bumps caches.
- Dashboard: placeholder widgets (counts) — real SEO widgets in L7-05.
- Tests: guests redirected, roles enforced, settings save bumps cache.

## Out of scope
- Domain resources (each domain task).

## Acceptance
- `/admin` fully Persian RTL with local fonts, zero external requests (check with `tools/shot.mjs` network log);
  `composer verify` green.
