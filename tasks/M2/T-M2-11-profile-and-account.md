---
id: T-M2-11
title: Profile and account endpoints
milestone: M2
type: backend
status: todo
depends_on: [T-M2-05, T-M2-06, T-M2-07, T-M2-08]
parallel_group: M2-E
touches: [backend-go/internal/profile, backend-go/internal/notify, backend-go/internal/http/routes_profile.go, backend-go/db/queries/profile, backend-go/contract/allowlist/profile.yaml]
skills: []
verify: cd backend-go && go test ./internal/profile/... ./internal/notify/... && make test-int PKG=./internal/profile/... && make contract ROUTES=profile
---

# T-M2-11 — Profile and account endpoints

## Why
Onboarding writes the profile and seeds the first period; everything cycle-related reads it. Read api-inventory §1.2,
domain-inventory §2 (User/UserProfile casts, `markRecalculated`, `hasCompletedProfile`) and the
`ProfileController` source (`syncOnboardingPeriodLog` :462, export, destroyAccount).

## Scope
1. `GET /profile`: `{user (with nested profile, as Laravel's loaded relation), profile, bmi}`; `BmiService`
   (`bmi_message` group from `message_contents` via a MessageContentRepository-compatible lookup — coordinate the
   interface with T-M2-17; a minimal read-only resolver here is fine).
2. `POST /profile`: unknown-field 422 with `allowed_fields`; rules and fa/en messages from `lang/*/profile.php`
   (controller 422 family, `profile.validation_failed` summary); derive `user_goal` from `pregnancy_intention`; seed
   `last_period_start = today` for non-pregnant users; `syncOnboardingPeriodLog` into `cycle_histories`
   (as `is_estimated` onboarding row — port exactly); `markRecalculated` (atomic `calculation_version` increment +
   status completed); Telegram notice on name change (`notify.Telegram`, no-op when env is missing); 422/500 catch
   bodies `{success:false,message,error}`.
3. `GET /profile/export`: raw model serializations (User hidden fields, decimal strings, plain `date` cast
   previous-day UTC for CycleHistory/Pregnancy/Reminder, `+03:30` for `exported_at` etc.).
4. `DELETE /account`: revoke all tokens, delete user (FK cascades), fa/en message.
5. Shared model serializers needed by other tasks (User, UserProfile) live in `internal/profile/model` with a stable
   API; document it in the package doc.

## Out of scope
Cycle recalculation logic itself (T-M2-13/15 read `calculation_version`).

## Acceptance
- `make contract ROUTES=profile` green, incl. onboarding sequences (POST profile → GET /cycle/period/history golden
  shows the seeded row — recorded by T-M2-05) and unknown-field / invalid-value 422s in fa and en.
- Integration test: concurrent `POST /profile` increments `calculation_version` exactly twice.
