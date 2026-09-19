---
id: T-M1-02
title: Backend — guaranteed year-long session tokens
milestone: M1
type: backend
status: todo
depends_on: [T-M1-01]
parallel_group: M1-B
touches: [backend/app/Providers/AppServiceProvider.php, backend/app/Http/Controllers/Api/V1/OtpAuthController.php, backend/config/passport.php, backend/docker/entrypoint.sh, backend/routes/api.php, backend/tests/Feature/Auth]
skills: [new-endpoint]
verify: cd backend && vendor/bin/pint --test && php artisan test --filter=Auth
---

# T-M1-02 — Backend — guaranteed year-long session tokens

## Why
Fix the server-side part of the root cause from `docs/investigations/session-logout.md`.

## Scope
- Every token issued at OTP verification expires **365 days** after issue; assert it in a test (read the token
  row's `expires_at`, and the JWT `exp`).
- Sliding session: an authenticated request made when the token has < 30 days left can obtain a fresh one-year
  token (e.g. `POST /api/v1/auth/refresh-session`, or whatever T-M1-01 recommends) without re-entering an OTP.
  Revoke the old token after issuing the new one.
- Make key/client bootstrap in `entrypoint.sh` provably idempotent (never regenerate keys or clients when they
  already exist; log loudly if keys are missing on a volume that has tokens).
- 401 responses stay JSON with a stable `error_code` (e.g. `token_expired` / `token_revoked` / `unauthenticated`) so the
  client can tell a real logout apart from any other 401.
- Scheduled `passport:purge` only removes tokens that are both expired **and** older than the grace window.

## Out of scope
Frontend storage (T-M1-03), Android shell (T-M1-04).

## Acceptance
- Feature tests: token lifetime = 365d; refresh extends and revokes the old one; revoked/expired tokens get the
  right `error_code`; admin block + account deletion still revoke all tokens.
- `pint --test` and `php artisan test` green.
