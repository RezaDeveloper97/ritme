# Progress log

One section per finished task (appended by `/site-task`).

## L0-01 — Scaffold Laravel 12 app, move design HTML, tooling and conventions
- Laravel **12.69** skeleton in `laravel-site/`; design export moved untouched to `design/html/` (29 pages, `assets/`,
  `wordpress/`, `README-WordPress.md`) — relative paths still work.
- Dev deps: Pest 3 (+ laravel, arch plugins), Larastan 3 (level 6), Pint (laravel preset + `declare_strict_types`,
  `strict_comparison`). Removed `laravel/sail` (not used on cPanel) and `axios` (YAGNI).
- `composer verify` = `lint` + `analyse` + `test` + `npm run build`. `tests/TestCase` calls `withoutVite()`.
- `.env.example`: `fa` locale, `APP_TIMEZONE=Asia/Tehran` (Iran has no DST since 2022 → no ambiguity), SQLite dev,
  commented MySQL block, `CACHE_STORE=file`, `SESSION_DRIVER=file`, `QUEUE_CONNECTION=database`.
- Placeholder `/` (`Route::view`, named `home`, Persian, noindex, no external fonts — Laravel's default welcome page
  pulled fonts.bunny.net and was replaced).
- `CLAUDE.md` (conventions), `docs/qa/README.md` (PNG screenshots untracked).
- Verify: pint ok, phpstan "No errors", pest 3 passed, vite build ok (empty-page CSS 3.75 KB gz).
