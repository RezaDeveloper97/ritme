---
id: T-M9-38
title: Messages read v6 facts (daily, selected, admin preview)
milestone: M9
type: backend
status: done
depends_on: [T-M9-32]
parallel_group: M9-C
touches: [backend-go/internal/messages/source.go, backend-go/internal/messages/manager, backend-go/internal/messages/v2.go, backend-go/internal/messages/rules, backend-go/internal/admin/messages/rules.go, backend-go/db/queries/messages, backend-go/contract/allowlist/messages.yaml, backend-go/api/openapi.yaml, backend-go/internal/messages/handlers.go, backend-go/internal/messages/v6_int_test.go, backend-go/internal/http/routes_messages.go, backend-go/internal/http/routes_admin_content.go]
skills: []
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/messages/... ./internal/admin/messages/... && make test-int PKG=./internal/messages/... && make contract ROUTES=messages
---

# T-M9-38 — Messages read v6 facts (daily, selected, admin preview)
**Model:** opus (health copy reaches users).

## Why
`/messages/daily` builds its own legacy engine over raw rows (wrapping cycle day: a late user is told "day N of a new
cycle"; PMS ≥ L−6), and the v2 facts use O+1 as fertile, never show lateness and give teens and pill users fertile
facts (gap 02 §5). DECISIONS.md I-11, D-130.

## Scope
In v6/shadow mode (v1 unchanged):
1. `StoreSource.Cycle` replaced by the state provider; delete `engineProfile`/`historyFromRow`.
2. Content phase via I-11: menstrual → menstruation; follicular + fertile_rising → follicular; high_fertility +
   ovulation_likely → ovulation; luteal + period_expected → luteal; `is_pms_window` = pms_possible days; `unknown` →
   neutral content only.
3. v2 facts: add `main_phase`, `subphase`, `days_late`, `data_quality`, `confidence`, `suggest_pregnancy_test`,
   `fertility_suppressed`; `days_to_period` from the engine (0 when late); `is_fertile_window` = display window; strip
   fertility facts when suppressed or in teen/menopause modes.
4. Template variables `pms_start_day` / `top_symptoms`: fill them from the engine/logs or remove them from the
   validator allow-list (`rules/validate.go:13`) — pick one, test it.
5. Admin rule preview uses the same facts.

## Tests
- Late 8 and 15 days → never "day N of a new cycle"; teen with goal TTC → no fertile facts; pill → no fertile facts;
  O+1 → not fertile.
- Agreement: daily message phase == home hero phase for 8 personas × 60 days.
- Messages goldens: v1 unchanged; v6 Go-native goldens.

## Out of scope
- The message engine v6 (M10).

## Acceptance
- Tests green; PROGRESS.md cites D-130.
