---
id: T-M9-32
title: cycle_view v6 on today/date with CYCLE_ENGINE flag and shadow diff
milestone: M9
type: backend
status: todo
depends_on: [T-M9-21, T-M9-26, T-M9-31, T-M9-31b]
parallel_group: M9-C
touches: [backend-go/internal/cycle/service, backend-go/internal/cycle/view, backend-go/internal/cycle/shadow, backend-go/internal/platform/config, backend-go/cmd/api/engine_shadow.go, backend-go/api/openapi.yaml, backend-go/contract/allowlist, backend-go/contract/golden-v6, backend-go/cmd/contract]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./internal/cycle/... ./cmd/... && make test-int PKG=./internal/cycle/service/... && make contract ROUTES=cycle,period
---

# T-M9-32 — cycle_view v6 on today/date with CYCLE_ENGINE flag and shadow diff
**Model:** opus.

## Why
This is the switch that makes users see v6: `GET /cycle/today` and `/cycle/date/:date` build `cycle_view` from the
engine (D-120), behind a per-environment flag so stage can compare before switching (X-03), keeping the legacy keys the
frontend still parses (X-05).

## Scope
1. Config `CYCLE_ENGINE = v1 | shadow | v6` (default `v1`; invalid → fail at start).
2. v6 path: auth rev → `state.For` → `Resolve(target, LocalToday)` + `FutureCycles` + `TodayUI` → §2.35 encoder; ETag
   helper; one query warm. Legacy keys rebuilt from the v6 status (X-05): `phase` via the I-11 mapping, `data_status`,
   `predictions` (window O−5..O, the single §2.29 confidence), `profile_values`/`calculated_values`/`effective_values`,
   `data_quality_details`, `daily_card` (from status + `today_ui`; fertility `unknown` on period_expected) and the per-day
   `calculation` block. Nulls where v6 has no value (the frontend tolerance lands in T-M9-40).
3. Shadow: serve v1 unchanged; after the response, a bounded worker pool (size 2, drop when full) computes v6 for the
   same request, classifies differences (`o_plus_one`, `lateness`, `warnings`, `confidence`, `phase`,
   `onboarding_anchor`, `suppression`, `new_fields_only`, `other`) into `engine_shadow_diff_total{class}` and keeps up to
   1 000 samples in Redis (`ritme-go:eng:shadow:samples`, TTL 7 d, uid hashed). `go run ./cmd/api engine-shadow-report`
   prints counts and samples.
4. Goldens: Laravel goldens keep running with `CYCLE_ENGINE=v1` (X-04). New Go-native goldens in `contract/golden-v6/`
   over the contract personas recorded from the Go app with `CYCLE_ENGINE=v6` (add a record mode to `cmd/contract` if
   missing), regenerated with `-update`; the §2.35 example persona checked by hand.
5. OpenAPI: v6 fields, `today_ui`, `limits`, nullable legacy fields; contract allow-list entries cite D-120.

## Tests
- Integration in each mode: v1 byte-identical to today (contract); v6 payload equals engine output for 5 personas;
  shadow serves v1 and records a classified diff for a late user (`lateness`) and an O+1 day (`o_plus_one`).
- Warm v6 `/cycle/today` = 1 query; 304 on repeat; write → new ETag.
- Legacy keys: the frontend's current zod schema (copied fixture) parses a v6 payload for the pill persona (nulls
  allowed after T-M9-40 — until then assert which fields are null and list them in PROGRESS.md).
- `X-Timezone` Berlin at 21:30Z → yesterday's Tehran date is not used.

## Budgets
- v6 warm p95 ≤ 15 ms, cold ≤ 60 ms (local, `internal/querybudget` updated to 1 / ≤ 5 queries).

## Out of scope
- `/cycle/month` (T-M9-33), `/home` (T-M9-34), deploys (T-M9-48).

## Acceptance
- Verify passes in v1 mode; v6 goldens committed; PROGRESS.md cites D-120.

## Note from the T-M9-31b security review (Low)
`Cache-Control: private, no-cache` lets the browser keep engine responses in its disk cache (shared-device risk after logout). When wiring ETags, pick one: send `Clear-Site-Data: "cache"` on logout, or have the frontend fetch engine GETs with `cache: "no-store"` and keep the ETag in memory. Never log `ETag` response headers (the rev is an activity signal).
