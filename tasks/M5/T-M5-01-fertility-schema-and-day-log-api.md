---
id: T-M5-01
title: Fertility — fertility_logs table, day log and today API (Go)
milestone: M5
type: backend
status: done
depends_on: []
parallel_group: M5-A
touches: [backend-go/db/migrations, backend-go/db/queries/fertility, backend-go/sqlc.yaml, backend-go/internal/fertility, backend-go/internal/http/routes_fertility.go, backend/database/migrations, backend-go/api/openapi.yaml, docs/go-migration/deviations.md]
skills: [new-endpoint]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/fertility/... && make test-int PKG=./internal/fertility/... && make schema-diff
---

# T-M5-01 — Fertility — fertility_logs table, day log and today API (Go)

## Why
The home tiles and «ثبت روز» (docs/fertility-ttc/README.md, `docs/design/ttc-v19/`) need LH test and cervical
mucus, which no column stores today; BBT, intercourse and symptoms already live in `daily_health_logs`.

## Scope
1. Goose migration + schema-only Laravel twin for `fertility_logs` (README). Take the next free goose number (M3/M4
   may have added some); `make schema-diff` green.
2. `internal/fertility`: merged day model; `GET/PUT /api/v1/fertility/days/{date}` — PUT writes lh/mucus to
   `fertility_logs` and bbt/intercourse/symptoms/note to `daily_health_logs` **through the Go healthlog service**
   (same side effects as `POST /health-logs`) in one transaction; explicit null clears a field.
3. `GET /api/v1/fertility/today`: tile values + chance (level/label/bars from the cycle view's `fertility_level`)
   + cycle day. ≤ 3 queries.
4. Validation with fa/en 422s; OpenAPI; `deviations.md` note (Go-only additions, M5).
5. Integration tests: round trip, clear via null, spotting triggers the same period-start side effect as health-logs,
   legacy `GET /health-logs/{date}` shows the BBT/intercourse written here.

## Acceptance
- Contract matches the README; schema-diff and tests green.

## Note from T-M5-04 (frontend contract assumptions)
Enums: LH `negative|faint|positive`; mucus `dry|sticky|creamy|egg_white`; intercourse `unprotected|protected`;
symptoms `ovarian_pain|bloating|breast_sensitivity|spotting`; chance level `none|low|medium|high|peak`; confidence
`low|medium|high`; evidence strength `strong|medium|none`. Day fields: `date, cycle_day, lh, mucus, bbt, bbt_time,
intercourse, symptoms[], note, chance`; PUT body uses the same snake_case keys, `null` clears a field.
See `frontend/src/entities/fertility/api/schema.ts` for tolerated alternatives.
