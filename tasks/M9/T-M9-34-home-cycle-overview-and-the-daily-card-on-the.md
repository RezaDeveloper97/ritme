---
id: T-M9-34
title: /home, cycle-overview and the daily card on the v6 engine
milestone: M9
type: backend
status: done
depends_on: [T-M9-32, T-M9-13]
parallel_group: M9-C
touches: [backend-go/internal/home, backend-go/contract/golden-v6/home, backend-go/contract/allowlist/home.yaml, backend-go/api/openapi.yaml]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/home/... && make test-int PKG=./internal/home/... && make contract ROUTES=home
---

# T-M9-34 — /home, cycle-overview and the daily card on the v6 engine
**Model:** opus.

## Why
`/home` uses the legacy engine (week calendar, `next_period` that never shows late, `cycle_prediction` O−5..O+1), its
health check calls 21–35 / 2–8 "normal" (a third standard; §2.40 forbids 21–45 as normal), `HasCycleData` means
"LMP set", and the page costs ≈ 19 queries (gap 02 §5, gap 07 §4.3). DECISIONS.md I-08, D-122.

## Scope
1. In v6/shadow mode: `week_calendar` from `State.Range`; `next_period` with `days_to_period` and `days_late`;
   `cycle_prediction` from `today_ui.prediction_card` (O−5..O, ovulation with source; null blocks under suppression);
   challenge cycle day from the v6 status (no wrap; days beyond the content range show no daily challenge — note it for
   the admin validator); `HasCycleData` = the engine has an anchor.
2. Health-check items from `clinical_flags` and `cycle_regularity` (I-08); remove the constants in
   `home/sections.go:922-1036`; copy reviewed by the main session (no "normal" for 21–45).
3. `/home/cycle-overview`: regularity = v6 enum (`regular|irregular|insufficient_data`), PMS = next − 3..next − 1, from
   the engine state; `support.go` digest from actual periods only.
4. Daily card rebuilt from status + `today_ui` (shared with T-M9-32).
5. Use the shared state provider and config caches; `LocalToday`; ETag on cycle-overview.
6. v1 mode unchanged.

## Tests
- Personas: late 8 days → `next_period.days_late = 8`, no rolled week; pill → no fertile dots; onboarding-only →
  `HasCycleData` true; no data → false; 18-day real cycles → health check "frequent" (never "normal").
- Week calendar == `cycle_view` per day (7 days × 8 personas).
- Query budget: `/home` ≤ 8 queries, cycle-overview ≤ 2 warm.

## Budgets
- `/home` warm p95 ≤ 40 ms, cold ≤ 120 ms.

## Out of scope
- Frontend home (T-M9-42/43), messages (T-M9-38).

## Acceptance
- Tests green; PROGRESS.md cites D-122.
