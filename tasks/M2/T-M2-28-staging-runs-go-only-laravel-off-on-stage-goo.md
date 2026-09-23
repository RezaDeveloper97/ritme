---
id: T-M2-28
title: Staging runs Go only (Laravel off on stage, goose owns stage schema)
milestone: M2
type: release
status: in_progress
depends_on: [T-M2-25]
parallel_group: M2-J
touches: [deploy/vhost-stage.inc,docker-compose.stage.yml,docker-compose.yml,deploy-stage.sh,.claude/skills/deploy-stage/SKILL.md,backend-go/cmd/api,backend-go/internal/platform/db/migrate.go,backend-go/internal/platform/config,docs/go-migration/cutover.md,docs/go-migration/migrations.md]
skills: [deploy-stage]
verify: ADMIN_SEED_PASSWORD=x docker compose -f docker-compose.yml -f docker-compose.stage.yml config -q && ADMIN_SEED_PASSWORD=x docker compose -f docker-compose.yml -f docker-compose.prod.yml config -q && cd backend-go && go test ./cmd/... ./internal/platform/...
---

# T-M2-28 — Staging runs Go only (Laravel off on stage, goose owns stage schema)

## Why
2026-09-23: the user decided staging switches to Go completely and development continues on Go from now on.
Production stays on Laravel until T-M2-26. Every API group already runs on Go in staging (T-M2-25); what still
reaches Laravel there is `/up`, `/storage/*` (non-group), `/docs`, `/oauth/*`, unported `/api/*`, the Blade
`/admin`, the Laravel `queue` worker, and Laravel's entrypoint migrations. The goose baseline equals the live
staging schema (`scripts/schema-diff.sh --against` staging dump: identical, 38 tables).

## Scope
1. `deploy/vhost-stage.inc`: `/up`, `/api/` (catch-all), `/storage/`, `/docs` → backend-go (use the existing
   request-time `$stage_backend_go_target` pattern so the shared prod proxy's `nginx -t` never depends on stage
   containers); `/oauth/` → 404 (unused by clients); `/admin` and `/admin/*` → 301 to `/panel/` (Blade retired on
   stage). Keep the gate, `/api/session/flag` → frontend, and the go-routes groups working. Prod-only files untouched.
2. Go owns stage migrations: backend-go runs goose on start when `RUN_MIGRATIONS=true` (default false in base, true
   in the stage overlay). On a DB that has Laravel's `migrations` table but no `goose_db_version`, it must
   `StampBaseline` instead of applying the baseline (never re-create existing tables). Log what it did. Tests.
3. `docker-compose.stage.yml`: Laravel `backend` and `queue` go under a compose profile (`laravel`) so a plain
   stage `up -d` no longer starts them; override backend-go's `depends_on` so it doesn't require `backend`
   (mysql + redis healthy only); backend-go `RUN_MIGRATIONS=true`. Passport keys already exist on the stage
   `backend-storage` volume (Go never generates keys) — document that a fresh stage volume would need them.
   Prod overlay/base behaviour unchanged (`config -q` for base, stage, prod).
4. `deploy-stage.sh`: build/start the Go-only set; after `up -d`, stop+remove the old Laravel stage containers
   (`ritme-stage-backend-1`, `ritme-stage-queue-1`) if still running; verification asserts `/up` and
   `/api/v1/languages` come from Go (`X-Backend: go`), `/admin` → 301 `/panel/`, `/oauth/…` 404.
5. Docs: deploy-stage skill, `cutover.md`, `migrations.md` (stage = goose-owned, prod = Laravel-owned until T-M2-27;
   schema changes still need both a Laravel and a goose migration until then).

## Acceptance
- Validations in `verify:` green; `nginx -t` in a local nginx:alpine with the real config files; routing test with
  stub upstreams for the paths above.
- No Laravel container is needed for staging to serve every path.
