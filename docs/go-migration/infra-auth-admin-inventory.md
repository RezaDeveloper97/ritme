# Infra, auth & admin inventory (Laravel `backend/`, captured 2026-09-23)

## 1. Auth — Passport v13.4.1 (league/oauth2-server 9.3.0, lcobucci/jwt 5.6.0)
Personal access tokens only (no password/refresh grants).
- Issue: `OtpAuthController::verifyOtp` → `$user->createToken('auth_token')->accessToken` (`:191`). `refreshSession` (`:220-250`) issues new then revokes old only within `refresh_window_days` (30). `logout` (`:212`) revokes current. Also revoked: admin block (`Admin/UserController.php:81`), account delete (`Api/V1/ProfileController.php:412`).
- Lifetime: `config/passport.php:74` `token_lifetime_days` 365 (`PASSPORT_TOKEN_LIFETIME_DAYS`), `:76` `refresh_window_days` 30; `AppServiceProvider.php:37-41` applies `P365D`.
- Keys: `storage/oauth-private.key` (PKCS#8, RSA 4096), `storage/oauth-public.key` (PKIX). Only on the `backend-storage` volume (excluded from `.dockerignore` and rsyncs). Private key 0600, owner www-data (uid 33).
- JWT (`AccessTokenTrait.php:63-76`): header `{"typ":"JWT","alg":"RS256"}` no kid; `aud` = personal client uuid as **string**; `jti` = 80 hex (`bin2hex(random_bytes(40))`) = `oauth_access_tokens.id`; `iat`,`nbf` float with µs (int if µs=0); `exp` = iat+365d same encoding; `sub` = user id string; `scopes` = `[]`.
- Laravel validation (`auth:api`, driver passport, provider users): Bearer header only → RS256 signature + `LooseValidAt` (no leeway) (`BearerTokenValidator.php:65-100`) → row `id=jti AND revoked=0` (`Bridge/AccessTokenRepository.php:64-67`; **DB `expires_at` not checked**) → `oauth_clients` by aud: exists, revoked=0, provider null or `users` (`Guards/TokenGuard.php:130-139`) → `User::find(sub)` else 401. `blocked_at` not checked (block revokes tokens).
- 401 (`bootstrap/app.php:75-107`): JSON `{message,error_code}`: `token_revoked` (jti missing/revoked), `token_expired` (expiry the **only** failed constraint), `unauthenticated` (else).
- `oauth_access_tokens`: `id char(80) PK`, `user_id` idx, `client_id uuid`, `name` (`auth_token`), `scopes text` (`[]`), `revoked bool`, timestamps, `expires_at`.

**Go recipe.** Verify: `x509.ParsePKIXPublicKey`; golang-jwt v5 `WithValidMethods(["RS256"])`, no leeway (NumericDate accepts floats; aud string or array) → `SELECT revoked FROM oauth_access_tokens WHERE id=?` → `SELECT revoked, provider FROM oauth_clients WHERE id=?` → load user by sub → map errors to the same codes (only signature-valid tokens failing solely on exp → `token_expired`). Issue: `x509.ParsePKCS8PrivateKey`, same claims, aud = latest non-revoked client whose `grant_types` contains `personal_access`; insert row `name='auth_token'`, `scopes='[]'`, `revoked=0`, `expires_at=now+365d`. Tokens are mutually valid across stacks.

## 2. OTP
- Table `otp_verifications`: id, mobile(11) idx, code(4) plain, expires_at, verified_at, attempts tinyint, timestamps.
- `config/sms.php`: 4 digits, 2 min expiry, 5 attempts, 60s resend. **No test/bypass mode by design** (`tests/Feature/OtpNoBypassTest.php`).
- send: regex `^09\d{9}$`; 429 + `retry_after` if created <60s ago; delete old rows; `random_int`; insert; dispatch `SendOtpSmsJob` in try/catch; never reveals `new_user`.
- verify: latest unverified row (none/expired 400); atomic attempt claim `UPDATE … WHERE attempts<5` (429); `hash_equals` (422); set `verified_at`; blocked → 403; firstOrCreate user, set `mobile_verified_at`, issue token.
- Throttles (`routes/api.php:41-44`): send 5/1, verify 10/1, refresh 10/1; redis cache store; key = IP (`trustProxies('*')` → XFF).
- Job (`Jobs/SendOtpSmsJob.php`): 3 tries, backoff [5,15], throws on SmsService false.
- Providers (`Services/Sms`): Kavenegar primary `GET https://api.kavenegar.com/v1/{key}/verify/lookup.json?receptor&token&template` (template `1507703`, success `return.status==200`); SMS.ir fallback `POST https://api.sms.ir/v1/send/verify` header `X-API-KEY`, body `{mobile,templateId,parameters:[{name:"OTPCODE",value}]}`, success `status==1`. Fallback list `['smsir']`.
- Queue: prod redis (`docker-compose.yml:66`), `queue` container `queue:work --tries=3 --sleep=3` (`:97-112`); local sync.

## 3. Admin panel (Blade — to be replaced by Go admin API + Next.js `admin-web/`)
- Mounted only when `ADMIN_PANEL_ENABLED=true` (`bootstrap/app.php:25-32`), prefix `/admin`, `web` + `setlocale:default`. **Same `backend` container** (`docker-compose.yml:80`); nginx splits by host: `adpanell.ritme.app` 404s `/api/` (`deploy/vhost-admin.inc`), `api.ritme.app` 404s `/admin` (`deploy/vhost-api.inc`).
- Auth: session guard `admin`, provider `admins` (`config/auth.php:49-52,78-81`); `Models/Admin.php` name, email, password (hashed), role super/editor, is_active, last_login_at. Login `Auth::attempt` + is_active + remember-me, session regenerate, throttle 5/min (`AuthController.php`). Middleware `admin.active` (logout inactive), `admin.super` (403) (`Middleware/EnsureAdminActive.php`, `EnsureSuperAdmin.php`). bcrypt `$2y$` cost 12 (Go verifies; writes `$2a$`, cost ≥12). Sessions redis, cookie `ritme-session`, APP_KEY-encrypted, PHP-serialized → not shareable. CSRF + `@method('PUT')`.
- Pages (`routes/admin.php`, 19 controllers ~2,350 LOC; 37 Blade files ~2,480 lines):
  - assets (`style.css`, fonts, `ckeditor/{path}` with traversal guard, `AssetController`; vendored `public/assets` ~1.5 MB)
  - login/logout, dashboard (counts + recent users), change own password
  - users: list (search, status filter), show (stats), update (name, subscription, goal), block/unblock (revokes tokens), delete (doesn't revoke — D-03)
  - CRUD + toggle: articles (CKEditor HTML, cover via `ImageOptimizer`), affirmations, challenges (+completions report), recommendations (subphase picker), banners (upload), task templates, info sections
  - CRUD: pregnancy weeks, phase contents
  - smart messages (`message_contents`): list, edit, update, approve, toggle
  - super only: languages (CRUD, toggle, regenerate, per-namespace translation editor writing JSON files), admins CRUD
  - shared multi-language form component `resources/views/components/admin/translatable.blade.php` (one input per active language)
- Uploads: banners `store('banners','public')` no optimisation (`BannerController.php:148-151`; jpeg/png/webp, 4 MB, ≥800×400). Articles `ImageOptimizer` (GD fit 1080×1080, WebP q82, `Str::random(40)` name; `exif` missing in image so auto-rotate is a no-op). Files in `storage/app/public/{banners,articles}`; URL `APP_URL/storage/<path>` (`config/filesystems.php:44`); DB stores relative `image_path`, API returns absolute `image_url`. PHP limits 10M/12M; nginx 25m.

## 4. Console / jobs / events
Only command `translations:import` (dev tool: copies `../frontend/messages/*` to `resources/translations`). No scheduler (CLAUDE.MD forbids `passport:purge`). Only job `SendOtpSmsJob`. No events/listeners/notifications/mail (mailer `log`), no FCM. `user_notifications` only read (`HomeController:323-375`). `TelegramNotifier` on name change (`ProfileController.php:497-505`) — `TELEGRAM_*` not in compose env → effectively disabled in prod. Pregnancy alerts synchronous.

## 5. Config & env
Compose (`docker-compose.yml:48-91`): APP_NAME, APP_ENV, APP_DEBUG, APP_KEY, APP_URL, LOG_CHANNEL=stderr, DB_CONNECTION=mysql, DB_HOST, DB_PORT, DB_DATABASE, DB_USERNAME, DB_PASSWORD, REDIS_CLIENT, REDIS_HOST, REDIS_PORT, CACHE_STORE=redis, SESSION_DRIVER=redis, QUEUE_CONNECTION=redis, SMS_PROVIDER, SMSIR_API_KEY, SMSIR_TEMPLATE_ID, SMSIR_LINE_NUMBER, KAVENEGAR_API_KEY, KAVENEGAR_SENDER, KAVENEGAR_TEMPLATE_LOGIN_OTP, RUN_MIGRATIONS, ADMIN_PANEL_ENABLED, ADMIN_SEED_EMAIL, ADMIN_SEED_PASSWORD, ADMIN_SEED_NAME, SWAGGER_USER, SWAGGER_PASSWORD, L5_SWAGGER_GENERATE_ALWAYS. Overlays: `docker-compose.prod.yml:37-49`, `docker-compose.stage.yml` (APP_ENV/DEBUG, PROXY_CONF, STAGE_CONF, DB_ROOT_PASSWORD, NEXT_PUBLIC_API_BASE_URL). Config: APP_TIMEZONE (default **Asia/Tehran**, `config/app.php:70`), APP_LOCALE, CORS_ALLOWED_ORIGINS, PASSPORT_TOKEN_LIFETIME_DAYS, PASSPORT_REFRESH_WINDOW_DAYS, TELEGRAM_BOT_TOKEN/CHAT_ID/TIMEOUT (`config/services.php:32-34`), SESSION_* (120 min, lax).
Cache redis: `LanguageRegistry` `rememberForever('languages.registry')`, `CycleEngineCache` 24h; PHP-serialized, prefixes `ritme-cache-`/`ritme-database-` → Go uses `ritme-go:`. Logging stderr. Disks `local` (`storage/app/private`), `public` (`storage/app/public`). Trusted proxies `*` (`bootstrap/app.php:42-48`). Health `/up`.

## 6. Deployment
- Image (`backend/Dockerfile`): composer stage → `php:8.4-apache` (pdo_mysql, bcmath, opcache, pcntl, intl, zip, gd jpeg/webp, phpredis), port 80.
- Entrypoint (`backend/docker/entrypoint.sh`): storage skeleton + chown; `RUN_MIGRATIONS=true` → `migrate --force` (10 retries); Passport keys only if both missing (refuse if one; warn if tokens exist); personal client only if count=0; `storage:link`; seeders Language/Admin/MessageContent/Recommendation; `l5-swagger:generate`; config/route/event cache.
- Compose: `mysql` (MariaDB 11.4 — CPU x86-64-v1), `redis` (AOF), `backend` (127.0.0.1:8080), `queue`, `frontend`, `proxy` (nginx, prod). Volumes `mysql-data`, `redis-data`, `backend-storage`. No backend healthcheck.
- nginx: upstream pinned to **container name** `ritme-backend-1:80` (`deploy/proxy-ssl.conf:14`) after the 2026-08-31 Docker-DNS-to-staging incident. Staging = project `ritme-stage`, no published ports, external `ritme-edge` network, aliases `stage-backend`/`stage-frontend`; one host path routing (`deploy/vhost-stage.inc:108-112`): `/api`, `/oauth`, `/storage`, `/admin`, `/docs` → stage-backend except `= /api/session/flag` → Next.js; `/up` exempt from Basic-auth gate.
- Deploy scripts assert: `/up` 200, `/api/v1/banners` 401, `api/admin` 404, `adpanell/admin/login` 200, `adpanell/api/...` 404, web 307 (`deploy.sh:184-193`, `deploy-stage.sh:158-168`).
- For Go: multi-stage → distroless/alpine, run as uid 33, mount `backend-storage` (read keys, translations, public uploads), serve `/storage/*`, `/up`, `/docs`; OTP job via asynq (in-process worker); no migrations/keygen/seeders in entrypoint (until cutover); compose healthcheck; own pinned container name.

## 7. Localization
`backend/lang`: en/cycle.php, en/profile.php, fa/cycle.php, fa/profile.php, fa/validation.php (`__('profile.*')`, cycle labels). DB `languages` + `message_contents` (see domain inventory). `TranslationStore`: `resources/translations/{fa,en}/*.json` (23 namespaces) deep-merged with `storage/app/translations/<code>/*.json`, default-language fallback; served at `GET /api/v1/languages`, `/languages/{code}/messages`. `LanguageProvisioner` writes `lang/<code>` into the image (bug D-04).

## 8. backend/CLAUDE.MD rules (to carry into backend-go/CLAUDE.md)
SOLID/DRY/KISS/YAGNI; versioned `/api/v1`; every endpoint documented (OpenAPI). Sessions: exactly 365-day tokens (never `P1Y`), refresh only in 30-day window (issue before revoke), 401 always JSON with `error_code` (never 401 for other errors — use 403/422/429), revoke only on logout/admin block/account delete/refresh, no purge, token never in a cookie (frontend sets `ritme_auth` flag), keys never shipped, rotation logs everyone out. Multi-language: languages are data (no hard-coded fa/en in new code), only default language required, admin forms use the translatable component.

Skills/tooling to update after the migration: `verify-all` (pint + artisan test), `new-endpoint` (FormRequest/Resource/Pint/actingAs), `deploy` (artisan seeds, entrypoint migrations, trustProxies, cors.php), `deploy-stage` (`/oauth`, `/storage`, manual seeders), `local-dev` (artisan serve, sqlite, sync queue, tinker OTP; wrongly claims separate admin container), `next-task` (points at `backend/CLAUDE.MD`), agents `backend-reviewer`, `security-auditor`, `test-runner`, `.claude/hooks/format.sh:14-15` (Pint).

## Risks for zero-downtime cutover
1. Token compatibility — same keys + client row + row format; exact 401 codes; float claims; don't add a DB `expires_at` check; clocks in sync (no leeway).
2. Key file access (0600 uid 33); Go never generates keys.
3. Admin passwords bcrypt `$2y$` cost 12 — portable.
4. Admin sessions not shareable → re-login.
5. File URLs `APP_URL/storage/...` — Go serves `/storage/*` from the same volume; keep `/storage` routed to the backend on stage and prod.
6. Migration ownership — only one owner (Laravel until cutover, then goose baseline); `languages` migration seeds data.
7. Queue drain before OTP sending moves.
8. Cache/rate-limit prefixes; per-IP throttle via trusted XFF.
9. Pinned upstream container name.
10. Behaviour parity: validation messages, envelopes, locale resolution, HtmlSanitizer, `profile_completed`.
11. Blocked users rely on revocation, not `blocked_at`.
12. Deploy-script status assertions must follow the split.

## Size estimates (dev-days)
Auth 3–5 · engines/cycle/pregnancy/messages/home 25–40 · admin (Go API + Next.js app) 15–22 · localization 3–4 · jobs 1 · config/deploy 2–3 · migrations baseline 1–2 · test port 8–12 · tooling 1.
