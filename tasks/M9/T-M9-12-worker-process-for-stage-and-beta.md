---
id: T-M9-12
title: Worker process for stage and beta
milestone: M9
type: backend
status: done
depends_on: [T-M9-11, T-M9-10]
parallel_group: M9-B
touches: [backend-go/cmd/api/main.go, backend-go/cmd/api/worker.go, backend-go/internal/auth/lifecycle.go, docker-compose.stage.yml, docker-compose.beta.yml, deploy-stage.sh, deploy-beta.sh, .claude/skills/deploy-stage/SKILL.md, .claude/skills/deploy-beta/SKILL.md]
skills: []
verify: cd backend-go && go vet ./... && go test ./cmd/... ./internal/auth/... && go build -o /dev/null ./cmd/api
---

# T-M9-12 — Worker process for stage and beta
**Model:** opus. Deployment files: stage and beta only; production compose is not touched.

## Why
Nightly snapshots, warmers and sweeps should not compete with request latency in the API container (gap 07 §7.4,
DECISIONS.md X-10). One binary, two roles keeps deployment simple.

## Scope
1. `api worker` subcommand: asynq server + scheduler + the pollers (labs, health-record extract, learning dispatch);
   no HTTP listener except the metrics listener on its own port.
2. `API_RUN_WORKERS` (default `1` for local dev and tests; `0` on stage/beta API containers): when 0, the API starts
   no asynq server, scheduler or pollers (it still dispatches).
3. `backend-go-worker` service in `docker-compose.stage.yml` and `docker-compose.beta.yml`: same image, same env and
   storage volumes as `backend-go`, command `worker`, restart policy, no published ports.
4. Deploy scripts and the deploy skills mention and health-check the worker. `deploy-beta.sh` and
   `.claude/skills/deploy-beta/SKILL.md` currently have uncommitted changes from other work: if either is still dirty
   when you start, do not edit it — stop and ask the user.

## Tests
- Unit: with `API_RUN_WORKERS=0` no lifecycle hook starts a server/scheduler/poller; with `1` they start.
- `go build ./cmd/api`; `docker compose … config` renders for stage and beta with the example env.
- After the next stage deploy (by the user or T-M9-48): an OTP SMS job is processed by the worker container (smoke
  step recorded in PROGRESS.md).

## Out of scope
- Production, autoscaling.

## Acceptance
- Worker role works locally (`go run ./cmd/api worker` processes a queued job); compose files render; deploy docs
  updated (or the dirty-file note recorded).
