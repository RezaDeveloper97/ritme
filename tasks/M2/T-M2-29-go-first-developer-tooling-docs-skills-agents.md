---
id: T-M2-29
title: Go-first developer tooling (docs, skills, agents, local dev)
milestone: M2
type: backend
status: done
depends_on: [T-M2-24]
parallel_group: M2-J
touches: [backend-go/CLAUDE.md,backend-go/README.md,backend/CLAUDE.MD,frontend/CLAUDE.md,.claude/skills/local-dev,.claude/skills/verify-all,.claude/agents/backend-reviewer.md,.claude/agents/test-runner.md,.claude/agents/security-auditor.md,.claude/hooks,tasks/README.md,docs/go-migration/README.md]
skills: [verify-all]
verify: cd backend-go && go vet ./... && go test ./...
---

# T-M2-29 — Go-first developer tooling (docs, skills, agents, local dev)

## Why
2026-09-23: the user decided development continues on Go (`backend-go/`) from now on; staging runs Go only
(T-M2-28). Laravel (`backend/`) stays in the repo only because production still runs it until T-M2-26/27 — it is
frozen: no new features there. The developer tooling still assumes Laravel.

## Scope
1. `backend-go/CLAUDE.md` + `README.md`: state that backend-go is THE backend for all new work; how to run it
   locally against a dev DB (goose migrations, seed data), contract harness usage, schema-change rule (until T-M2-27:
   goose migration + mirror Laravel migration so prod keeps working).
2. `backend/CLAUDE.MD`: a banner at the top: frozen, maintenance-only for production until cutover; new work → backend-go.
3. `frontend/CLAUDE.md`: only the backend/API references (e.g. local API URL, where endpoints live) — no other edits.
4. `.claude/skills/local-dev`: run backend-go locally (test-stack MariaDB/Redis from `backend-go/docker-compose.test.yml`
   or a dedicated dev DB, `make run`, goose migrate + seed), frontend `.env.local` pointing at it, and how to log
   in without SMS (`SMS_PROVIDER=log` + read the OTP from the DB). Keep the Laravel recipe as a short fallback.
5. `.claude/skills/verify-all`: Go first; Laravel checks only as "legacy (prod) — run when touching backend/".
6. `.claude/agents/backend-reviewer.md` → reviews Go (Fiber v3, sqlc, goose, PHP-compat rules, contract/OpenAPI
   discipline); `test-runner.md`, `security-auditor.md`: add backend-go commands/paths.
7. `.claude/hooks`: make sure Go formatting/lint hooks are active for backend-go; Laravel pint hook stays for backend/.
8. `docs/go-migration/README.md`: status line (staging on Go, prod on Laravel, dev on Go).

## Acceptance
- `verify:` green; every doc/skill references commands that exist (`make` targets, scripts) — check them.
- Do not edit `tasks/README.md` or `tasks/INDEX.md` (another session is editing them).
