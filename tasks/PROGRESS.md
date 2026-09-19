# Progress notes

One `## T-Mx-NN` section per finished task: what shipped, commands/env vars, migrations, open items.

## T-M1-01 — Diagnose premature logout
- Shipped `docs/investigations/session-logout.md` (H1–H7 with evidence). Server-side lifetime ruled out (DB `expires_at` and JWT `exp` = 365 d; keys stable since 2026-08-16; no stray revokes).
- Root causes: (1) `ritme_auth` flag cookie written via `document.cookie` → WebKit 7-day cap, and SessionGuard keeps a user with a valid localStorage token on `/signup`; (2) iOS storage partitioning (Safari tab vs Home Screen app vs in-app browser) — confirmed in prod logs; (3) prod bundle v1.0.2 still clears the token on any 401 (latent).
- Scope edits: T-M1-02 → hardening (401 `error_code`, stop shipping laptop Passport keys via rsync/`COPY`, fix entrypoint creating a 2nd personal client; do NOT rotate keys); T-M1-03 → main fix; T-M1-04 → optional flush + manual tests.
- Open: proxy logs only since 2026-09-07; nginx `log_format` lacks `$host`; real-iPhone check of the 7-day cap pending (T-M1-03).
