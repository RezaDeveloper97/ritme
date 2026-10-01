---
id: CB-MENO-01b
title: Checkups audience filter and menopause checkup activation
epic: MENO
type: backend
status: done
depends_on: [CB-MENO-01]
parallel_group: MENO-A
touches: [backend-go/internal/checkups,backend-go/internal/admin/checkups,backend-go/db/queries/checkups,backend-go/db/queries/admin,backend-go/api/openapi.yaml,backend-go/contract,backend-go/db/migrations,backend/database/migrations,backend-go/internal/i18n/testdata]
skills: [new-endpoint]
boards: [nbl_Meno_Checkups.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/checkups/... ./internal/admin/checkups/... && make test-int PKG=./internal/checkups/... && make test-int PKG=./internal/admin/checkups/... && golangci-lint run && make schema-diff
---

# CB-MENO-01b — Checkups audience filter and menopause checkup activation

## Why
CB-MENO-01 added `checkup_types.audiences` (JSON list of life modes, NULL = everyone) and seeded 9 `meno_*` checkup rows **inactive**, because neither the checkups engine nor the admin checkup-types API reads `audiences` yet.

## Boards
- `nbl_Meno_Checkups.dc.html` (if absent in the snapshot, use the menopause home/checkups sections)

## Scope
1. Checkups engine (`internal/checkups`): list/plan only checkup types whose `audiences` is NULL or contains the user's current life mode (bloom `user_life_profiles`).
2. Admin checkup-types API (`internal/admin/checkups`): read/write `audiences` with validation against the known modes; OpenAPI + contract.
3. Goose migration (number agreed with the bloom session first) activating the 9 `meno_*` rows; Laravel schema-only twin if schema-diff needs it.

## Out of scope
- admin-web form field (CB-MENO-04); frontend checkups screen (CB-MENO-09).

## Acceptance
- A cycle-mode user does not see `meno_*` rows; a menopause-mode user does; existing shared rows unchanged for everyone (int tests).
- `verify` green.
