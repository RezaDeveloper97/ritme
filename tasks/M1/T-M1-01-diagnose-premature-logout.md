---
id: T-M1-01
title: Diagnose premature logout (root cause, with evidence)
milestone: M1
type: investigate
status: todo
depends_on: []
parallel_group: M1-A
touches: [docs/investigations/session-logout.md]
skills: []
verify: test -s docs/investigations/session-logout.md
---

# T-M1-01 — Diagnose premature logout (root cause, with evidence)

## Why
Users get signed out after ~1–2 days. The goal is a session that lasts **1 year** and only ends on an explicit
sign-out, account deletion or admin block. The backend already sets
`Passport::tokensExpireIn / refreshTokensExpireIn / personalAccessTokensExpireIn(now()->addYear())`
(`backend/app/Providers/AppServiceProvider.php:35`), so the lifetime setting alone is **not** the cause. Find the
real one before changing code — no speculative fixes in this task.

## Scope
Check every path that can end a session and record evidence (command + output) for each:
1. **Token itself** — on prod (see memory `ritme-server-deployment`), read `expires_at` of recent rows in
   `oauth_access_tokens` and decode one JWT's `exp`. Confirm the year-long expiry is actually applied to
   `createToken()` personal tokens (Passport version behaviour) and that `revoked` isn't flipped by anything.
2. **Signing keys** — `storage/oauth-*.key` on the named volume: are they stable across `deploy.sh` runs
   (compare fingerprints / mtime vs. deploy dates)? Does `backend/docker/entrypoint.sh` ever create a new
   personal-access client or keys? Is `APP_KEY` stable?
3. **401 sources** — grep nginx/proxy + Laravel logs for 401s on `/api/v1/*`; find which endpoints return JSON 401
   to valid users. The frontend clears the token on **any** JSON 401
   (`frontend/src/shared/api/apiClient.ts:55`).
4. **Client storage** — `localStorage` token + `ritme_auth` flag cookie set via `document.cookie`
   (`frontend/src/shared/session/token.ts`): iOS Safari ITP (7-day cap on script-written cookies and
   script-writable storage in browser tabs), home-screen PWA vs. Safari tab storage separation, Android WebView
   shell (`android-shell/.../MainActivity.kt`), origin changes (http→https, domain moves) that hide the token.
5. **SessionGuard** repair logic (`frontend/src/shared/session/SessionGuard.tsx`): any path that clears a valid session
   (e.g. cookie missing + token present on a resume, `visibilitychange` race).
6. Anything else found on the way (service worker serving stale HTML, forced-update flow, etc.).

## Out of scope
Any fix. Write the findings; T-M1-02/03/04 implement.

## Acceptance
- `docs/investigations/session-logout.md` lists each hypothesis as **confirmed / ruled out / unknown**, with the
  evidence for each, the root cause(s) ranked, and the concrete fix recommendation mapped to T-M1-02 / 03 / 04.
- If the findings show T-M1-02/03/04 need different scope, edit those task files and say so in the report.
