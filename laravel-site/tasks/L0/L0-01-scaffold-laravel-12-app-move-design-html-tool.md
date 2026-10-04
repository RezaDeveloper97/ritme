---
id: L0-01
title: Scaffold Laravel 12 app, move design HTML, tooling and conventions
milestone: L0
type: setup
status: done
depends_on: []
parallel_group: L0-A
touches: [composer.json,package.json,app,bootstrap,config,database,routes,tests,public,resources,lang,design,CLAUDE.md,.gitignore,.editorconfig,.env.example,phpstan.neon,pint.json]
skills: []
verify: composer verify
---

# L0-01 — Scaffold Laravel 12 app, move design HTML, tooling and conventions

## Why
Everything else builds on a clean Laravel skeleton with the quality gates (`composer verify`) in place from day one.

## Scope
- Move the design export out of the app root, untouched: `*.html`, `assets/`, `wordpress/`, `README-WordPress.md` →
  `design/html/` (keep relative paths working so the pages still open in a browser). This is the **fidelity source**.
- `composer create-project laravel/laravel` (12.x) in a temp dir and move it into `laravel-site/` (don't clobber
  `design/`, `tasks/`).
- `.env.example`: `APP_LOCALE=fa`, `APP_FALLBACK_LOCALE=fa`, `APP_TIMEZONE=Asia/Tehran`, `DB_CONNECTION=sqlite` for dev,
  commented MySQL block for production, `CACHE_STORE=file`, `QUEUE_CONNECTION=database`, `SESSION_DRIVER=file`.
- Dev deps: Pest 3 (+ arch plugin), Larastan (level 6 to start, raise later), Pint (Laravel preset), `laravel/pail` OK.
- Composer scripts: `lint` (pint --test), `analyse` (phpstan), `test` (pest --parallel when available), `verify`
  (lint + analyse + test + `npm run build`). `npm run build` must succeed (default Vite for now).
- `laravel-site/CLAUDE.md`: conventions summary pointing to `tasks/README.md` decisions (domain layout, cache-aside
  rule, SEO checklist per page, "no externals", Persian copy red lines, how to run `composer verify`). Keep it short.
- `.gitignore`: vendor, node_modules, .env, storage artefacts, `public/build`, `public/media`, `public/sw.js`, `docs/qa/**/*.png` heavy outputs allowed? — keep QA PNGs **out** of git (write `docs/qa/README.md` explaining).
- Sanity route `/` returns 200 with Laravel default (replaced in L3).

## Out of scope
- Any page conversion, Tailwind, admin.

## Acceptance
- `design/html/index.html` opens in a browser with styles/fonts.
- `php artisan serve` works; `composer verify` green; `git status` shows only `laravel-site/` paths.
