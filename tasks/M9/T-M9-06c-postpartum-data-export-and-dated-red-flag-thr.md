---
id: T-M9-06c
title: Postpartum data export and dated red-flag thresholds in the daily message
milestone: M9
type: backend
status: blocked
depends_on: [T-M9-06b]
parallel_group: M9-A
touches: [backend-go/db/queries/postpartum/export.sql,backend-go/internal/postpartum/store,backend-go/internal/accountexport,backend-go/internal/messages/postpartum_source.go,backend-go/contract/golden/profile,backend-go/contract/allowlist/profile.yaml,docs/go-migration/deviations.md]
skills: []
verify: cd backend-go && make sqlc && go vet ./... && go test ./... && make contract ROUTES=profile,postpartum
---

# T-M9-06c — Postpartum data export and dated red-flag thresholds in the daily message
**Model:** opus (health data, privacy).

## Why
Split out of T-M9-06b: its export item lived outside its touches.

Today the account export (`internal/accountexport`, wired in `internal/http/routes_profile.go`) exports no postpartum data at all: not `postpartum_profiles`, not `epds_checks`, not `postpartum_recovery_logs`. Separately, `internal/messages/postpartum_source.go` builds red flags with `sig.Recovery.Alerts()`, which uses the version-1 thresholds (zero day). So a later `postpartum.redflags.fever_c` config version is ignored by the daily message.

## Needs a user decision first
- Adding a postpartum key to the export goes past the nine top-level keys approved in D-96. Either extend D-96 or add a new deviation.
- EPDS answers: D-54 says they are "never returned". Should the export include them (it is the user's own data), only the totals, or nothing?

## Scope
1. `db/queries/postpartum/export.sql` (+ `make sqlc`): the user's postpartum profile, recovery logs, and EPDS per the decision.
2. A `postpartum` section in `internal/accountexport` (`Sections`, `health.go`), plus a test that includes the recovery logs.
3. Re-record the profile `export*` goldens (Go-only) and update the allow-list with the deviation id.
4. In `postpartum_source.go`, use `AlertsOn(date)` with the log's date, so dated config versions apply; add a test with a later `fever_c` version.

## Out of scope
- The v1/v2 pregnancy alert double-count in `/pregnancy/alerts/summary` (separate product question).

## Acceptance
- The export test includes the recovery logs (and EPDS per the decision).
- The daily message uses the threshold version in force on the log date.
- verify green.
