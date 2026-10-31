---
id: T-M10-10
title: Challenges through the message engine: eligibility, contraindications, phase start
milestone: M10
type: backend
status: done
depends_on: [T-M10-06]
parallel_group: M10-B
touches: [backend-go/internal/challenges/program.go, backend-go/internal/challenges/service.go, backend-go/internal/challenges/handlers.go, backend-go/internal/challenges/eligibility.go, backend-go/internal/home/challenge.go, backend-go/internal/messages/feed/challenge.go, backend-go/internal/admin/challenges, backend-go/db/queries/challenges, backend-go/internal/challenges/store, backend-go/internal/admin/content/challenges_v2.go, backend-go/internal/home/sections.go, backend-go/internal/messages/facts, backend-go/internal/challenges/lang, backend-go/internal/messages/feed/feed.go, backend-go/internal/messages/feed/json.go, backend-go/internal/http/routes_messages.go, backend-go/db/queries/messages/facts.sql, backend-go/internal/messages/store, backend-go/internal/querybudget, backend-go/internal/messages/v2.go, backend-go/internal/home/context.go, backend-go/internal/http/routes_home.go, backend-go/internal/admin/messages/testdata, backend-go/api/openapi.yaml]
skills: []
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/challenges/... ./internal/messages/... ./internal/home/... && make test-int PKG="./internal/challenges/... ./internal/home/..." && make contract ROUTES=home
---

# T-M10-10 — Challenges through the message engine: eligibility, contraindications, phase start
**Model:** opus (contraindications are a safety rule).

## Why
SPEC §5.5: eligibility and contraindications are rules over the same facts, contraindications always win (e.g. hard
exercise in pregnancy or before 6 weeks postpartum), `start_rule` is `on_join` or `on_phase_start`, and the suggestion
comes through the "today's challenge" slot. Today eligibility is include/exclude lists of mode and phase only
(`challenges/program.go:34-68`), so "< 6 weeks postpartum" cannot be expressed; only a menstrual start exists; the home
uses a legacy pick and the `daily_challenge` placement is never served (gap 03 §2.8). D-167.

## Scope
1. `challenge_programs.eligibility` new shape `{all_of, any_of, exclude}` over the fact registry; the legacy
   include/exclude lists are read and converted on the fly (no data migration); the admin API writes the new shape.
   `contraindications` (T-M10-02 column) uses the same shape and is evaluated first; any match refuses with reason
   `contraindicated`.
2. `start_rule`: `on_join` (= today's `anytime`), `on_phase_start` + `start_phase` (any v6 `main_phase`; join only on
   the first day of that phase per the engine state), `fixed_date` kept.
3. Feed `daily_challenge` (T-M10-04 slot): an ongoing joined challenge wins until completed or abandoned; otherwise
   eligible, not-joined, active challenges are product candidates (priority = admin sort order); admin rules that test
   `challenge_id` can promote one. The legacy home pick (`home/challenge.go`) is used only when the feed has no
   candidate.
4. Stats: day-by-day drop-off (share of starters active on day n) next to participants and completion rate.

## Tests
- Pregnancy + a hard-exercise challenge whose eligibility includes pregnancy → refused (`contraindicated`);
  postpartum week 5 refused, week 6 allowed.
- `on_phase_start(luteal)` joinable on the first luteal day (engine fixture), refused on the second.
- Legacy eligibility rows behave exactly as before (table over the current fixtures).
- The feed keeps a joined challenge until completion; drop-off math on a fixture.

## Out of scope
- Admin-web challenge editor (T-M10-13).

## Acceptance
- Verify passes; PROGRESS.md cites D-167.
