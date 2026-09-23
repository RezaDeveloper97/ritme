---
id: T-M2-20
title: Admin API I — admin auth, roles, dashboard, users, admins
milestone: M2
type: backend
status: done
depends_on: [T-M2-08]
parallel_group: M2-E
touches: [backend-go/internal/admin/auth, backend-go/internal/admin/users, backend-go/internal/admin/admins, backend-go/internal/admin/dashboard, backend-go/internal/admin/httpadmin, backend-go/internal/http/routes_admin_core.go, backend-go/db/queries/admin/core.sql, docs/go-migration/admin-api.md]
skills: [security-review]
verify: cd backend-go && go test ./internal/admin/... && make test-int PKG=./internal/admin/...
---

# T-M2-20 — Admin API I: admin auth, roles, dashboard, users, admins

## Why
The Blade admin is replaced by a Next.js app (`admin-web/`, T-M2-22/23) that talks to a Go admin API. This task builds
the secure core. Read infra-auth-admin-inventory §3 and `app/Http/Controllers/Admin/{Auth,Dashboard,User,Admin,Account}Controller.php`.

## Scope
1. Design doc `docs/go-migration/admin-api.md`: base path `/api/admin/v1`, served **only** on the admin host
   (`adpanell.ritme.app`; the API host must 404 it, like today's `/admin` split), JSON envelope `{success,data,message}`,
   error shape, pagination shape, list/filter conventions, upload conventions (used by T-M2-21/22/23).
2. Admin auth: `POST /auth/login` (email + password, bcrypt `$2y$`/`$2a$` verify, `is_active` required, throttle 5/min
   per IP+email, updates `last_login_at`), `POST /auth/logout`, `GET /auth/me`. Session = opaque random id in Redis
   (`ritme-go:admin-session:*`, sliding 120 min, "remember me" 30 days), set as `HttpOnly; Secure; SameSite=Lax`
   cookie scoped to the admin host; CSRF via double-submit token header (`X-CSRF-Token`) for all mutating requests;
   CORS allows only the admin-web origin with credentials.
3. Middleware: `RequireAdmin` (session valid and admin still active → otherwise 401 and session destroyed),
   `RequireSuper` (403).
4. Endpoints: dashboard counts + recent users; own password change; users list (search by name/mobile, status
   filter, pagination), show (stats as in Blade), update (name, subscription, goal), block/unblock (block revokes all
   tokens), delete (**also revokes tokens** — D-03, confirm with user before implementing); admins CRUD (super only,
   cannot deactivate/delete self, new hashes bcrypt cost 12).
5. Audit log lines (slog) for every admin mutation (admin id, action, target).

## Out of scope
Content CRUD, uploads, languages (T-M2-21). The Next.js UI (T-M2-22/23).

## Acceptance
- Tests: login with a real Laravel-created bcrypt hash succeeds; inactive admin rejected; CSRF missing → 419/403;
  editor → super-only endpoint 403; session expiry; block revokes `oauth_access_tokens` rows.
- `security-review` pass over the diff with no High findings open.
