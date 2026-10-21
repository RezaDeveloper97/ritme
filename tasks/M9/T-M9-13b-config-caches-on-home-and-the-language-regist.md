---
id: T-M9-13b
title: Config caches on /home and the language registry, query budgets
milestone: M9
type: backend
status: done
depends_on: [T-M9-13,T-M9-08]
parallel_group: M9-B
touches: [backend-go/db/queries/home,backend-go/internal/home,backend-go/internal/i18n/registry.go,backend-go/internal/querybudget]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/home/... ./internal/i18n/... ./internal/querybudget/... && make test-int PKG="./internal/home/... ./internal/i18n/..."
---

# T-M9-13b — Config caches on /home and the language registry, query budgets

**Model:** opus.

## Why
T-M9-13 built `internal/platform/configcache` (lock-free snapshots, pub/sub `ritme-go:config-changed`, 60 s TTL) and moved
messages, recommendations, modules, plus and challenges onto it, but three parts of its scope sat outside its `touches`:
the `/home` config reads, the language registry and the query budgets (T-M9-08 had not built `internal/querybudget` yet).
Admin writers for every table below already publish (`internal/admin/httpadmin/publish.go`).

## Scope
1. `/home`: task templates, affirmations, active challenges and home articles read from configcache snapshots. Make the
   `db/queries/home` reads whole-table (active rows) and filter by phase/mode in Go; call sites
   `internal/home/sections.go` (~313, ~594, ~802) and `internal/home/challenge.go` (~78). (`internal/home/articles.go` named in
   T-M9-13 does not exist — find the real article read.)
2. `internal/i18n/registry.go`: the language list comes from a configcache snapshot of `languages` (no Redis GET + decode per
   request); keep `Flush` working (it becomes `Changed("languages")`). Then remove the now-redundant `content.RegistryTTL`
   middleware and `languageRegistryTTL` in `internal/http/routes_content.go` (T-M9-03 open item) — add that file to touches
   when you start.
3. Query budgets (`internal/querybudget`, from T-M9-08): `/messages/selected` constant in the number of rules; `/home` budget
   lowered to what step 1 achieves. Update `docs/engine-v6/perf-baseline.md`.

## Tests
- Integration: an admin write to each table is visible on the next request (same process) and in a second Hub via pub/sub.
- `/home` and `/messages/selected` query counts asserted through the querybudget harness.
- Language registry: `Flush` still invalidates; no Redis read on the hot path.

## Out of scope
- New caches beyond these tables; message engine v6 (M10).

## Acceptance
- `/home` config tables and the language list are served from snapshots; budgets recorded and enforced; verify green.

> Note (main session, 2026-10-21): `docs/engine-v6/perf-baseline.md` was dropped from touches so this task can run next to T-M9-22. Report the numbers in the task report; the main session adds them to the doc.

## Waiting for Docker (2026-10-21)
The code is done and the unit tests, vet and lint are green. The main session also removed `content.RegistryTTL`: `routes_content.go`, `content.go` and `content_test.go`. The disk filled up and Docker Desktop hung, so these steps still have to run before the commit:
1. `make test-int PKG="./internal/home/... ./internal/i18n/..."`
2. `make test-int PKG="-v ./internal/querybudget/..."`: read the `/api/v1/home` row and the `TestMessagesSelectedConstantInRules` line, then set the `home` row in `internal/querybudget/budgets.go` to the measured numbers (it is 20/15 today; warm is expected to be about 11). Watch for a cold count rising by 1–2 because the language snapshot is reloaded on every cold call.
3. `make test-int PKG=./internal/admin/...` (optional).
4. Add the before/after counts to `docs/engine-v6/perf-baseline.md`.
