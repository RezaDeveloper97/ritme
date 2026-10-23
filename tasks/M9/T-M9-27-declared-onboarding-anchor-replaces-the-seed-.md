---
id: T-M9-27
title: Declared onboarding anchor replaces the seed period row
milestone: M9
type: backend
status: done
depends_on: [T-M9-01, T-M9-24, T-M9-25]
parallel_group: M9-C
touches: [backend-go/internal/profile/service.go, backend-go/internal/profile/onboarding.go, backend-go/db/queries/profile, backend-go/db/migrations, backend-go/internal/cycle/inputs, backend-go/api/openapi.yaml, backend-go/contract/allowlist]
skills: []
verify: cd backend-go && make sqlc && make schema-diff && go vet ./... && go test ./internal/profile/... ./internal/cycle/... && make test-int PKG="./internal/profile/... ./internal/cycle/inputs/... ./db/migrations/..." && make contract ROUTES=profile,onboarding,cycle,home
---

# T-M9-27 — Declared onboarding anchor replaces the seed period row
**Model:** opus (health data). Runs alone in a batch (data migration).

## Why
Onboarding writes the declared LMP into `cycle_histories` as a confirmed `user_profile_confirmed` row with an assumed
end (`profile/service.go:463-513`): an assumed end stored as real (§2.2, §2.40), a declared length feeding the medians
(§2.9), and §2.41 (stale anchor) can never fire (gap 02 S1–S3). DECISIONS.md I-01.

## Scope
1. `profile.Save` / onboarding write `onboarding_last_period_start` whenever the declared LMP is set or changed; nothing
   else writes it. `last_period_start` keeps its legacy meaning (latest actual start, still written by Save for legacy
   readers).
2. Delete `syncOnboardingPeriodLog` and the `InsertOnboardingCycleHistory` / `UpdateOnboardingCycleHistory` queries.
3. Goose **data** migration (no schema change; Go-only — production runs Laravel and is not touched):
   - untouched seed (start = profile LMP, end/bleeding_length = what a re-sync would write) → copy the start into
     `onboarding_last_period_start` and delete the row;
   - edited seed (differs from the re-sync output) → `source = user_logged`, `is_confirmed = 1` (the user said it is real);
   - bump `user_revisions` for every affected user in the same migration;
   - `down`: re-create the deleted seeds from the column with the old re-sync rule; converted rows stay `user_logged`
     (documented).
4. `inputs.Load` drops the pre-migration seed fallback once the migration exists (keep it only behind a test-only
   path if needed).
5. Visible change, in v1 mode too: onboarding-only users now resolve `onboarding_based` (the seed used to make them
   `user_logged`) and the calendar no longer paints the declared period as logged. Add contract allow-list entries
   citing D-123 for the affected profile/cycle/home cases; do not re-record Laravel goldens.

## Tests
- Migration integration test with 4 personas: untouched seed; seed edited by the user; seed followed by later logged
  periods; user without a seed. Up → expected rows/column; down → seeds back.
- Profile integration: a declared LMP creates no `cycle_histories` row; changing it updates only the column.
- Engine via `inputs.Load`: onboarding-only user → `onboarding_declared` anchor; 90 days later → stale anchor (§2.41).
- `make contract ROUTES=profile,onboarding,cycle,home` green with the new allow-list entries.

## Out of scope
- Calendar rendering of the assumed range (T-M9-33, T-M9-45).

## Acceptance
- Tests green; PROGRESS.md cites D-123 and notes the deploy is user-visible (approved in T-M9-01).
