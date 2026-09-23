---
id: T-M2-27
title: Decommission Laravel, hand migrations to goose, update skills and docs
milestone: M2
type: release
status: todo
depends_on: [T-M2-26]
parallel_group: M2-L
touches: [backend, backend-go, docker-compose.yml, docker-compose.stage.yml, docker-compose.prod.yml, deploy, .claude/skills, .claude/agents, .claude/hooks, CLAUDE.md, docs/go-migration, tasks/README.md]
skills: [verify-all, deploy-stage]
verify: test ! -d backend-go && cd backend && go vet ./... && go test ./... && make contract ROUTES=all
---

# T-M2-27 — Decommission Laravel and update tooling

## Why
After the 7-day observation (T-M2-26) Laravel is dead weight: two stacks, PHP in the images, stale skills. Finish the
migration cleanly. Needs user confirmation before deleting `backend/` and before the production deploy.

## Scope
1. Migrations: stamp goose version on stage and prod (documented procedure from T-M2-03), make the Go entrypoint run
   goose (`RUN_MIGRATIONS`), port the boot-time idempotent seeders (languages, admin from `ADMIN_SEED_*`,
   message contents from the embedded defaults, recommendations) and the one-off content seeders as Go commands.
2. Remove `backend/` (Laravel), the `queue` container, PHP from compose; `git mv backend-go backend`; fix every path
   (Makefile, Dockerfile, compose, deploy scripts, contract harness now records goldens from Go, enumgen deleted).
   Keep service/container names and the pinned nginx upstream consistent (`ritme-backend-1` now = Go).
3. Drop Laravel-only tables only if the user agrees (sessions, cache, cache_locks, jobs, job_batches, failed_jobs,
   password_reset_tokens, oauth_auth_codes, oauth_refresh_tokens, oauth_device_codes) — **keep** `oauth_access_tokens`
   and `oauth_clients`.
4. Tooling: rewrite skills `verify-all`, `new-endpoint` (already Go from T-M2-19 — remove the Laravel note), `deploy`,
   `deploy-stage`, `local-dev` (Go + MariaDB docker, OTP from DB, admin-web), `next-task` (backend rules file path);
   agents `backend-reviewer` (Go idioms + contract rules), `security-auditor`, `test-runner`; hooks (`format.sh` →
   gofmt only, `style-gate.sh`/`protect-files.sh` paths); root and backend `CLAUDE.md`; memory notes that mention
   Laravel (ritme-server-deployment, local-dev, admin panel, backend testing gotchas, cycle engine).
5. Final prod + stage deploy through the updated skills; deploy assertions updated (`/up`, `/api/v1/banners` 401,
   admin host routes).
6. `docs/go-migration/README.md` → "Completed" summary; tasks/README.md milestone note.

## Out of scope
New features; fixing preserved quirks (open product tasks for D-05 etc.).

## Acceptance
- No PHP left in the repo or images (except historical docs); `verify-all` green; stage and prod deployed via the
  updated skills with all assertions passing.
- `/next-task`, `/local-dev`, `/deploy-stage`, `/verify-all` work end to end on the Go stack.
