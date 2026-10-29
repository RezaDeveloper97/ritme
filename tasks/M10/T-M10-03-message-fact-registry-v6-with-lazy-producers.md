---
id: T-M10-03
title: Message fact registry v6 with lazy producers
milestone: M10
type: backend
status: done
depends_on: [T-M10-02]
parallel_group: M10-B
touches: [backend-go/internal/messages/rules/v6.go,backend-go/internal/messages/rules/v6_test.go,backend-go/internal/messages/store,backend-go/internal/messages/facts_int_test.go,backend-go/internal/messages/facts, backend-go/internal/messages/v2.go, backend-go/internal/messages/source.go, backend-go/internal/messages/rules/validate.go, backend-go/internal/admin/messages, backend-go/internal/http/routes_messages.go, backend-go/db/queries/messages, backend-go/api/openapi.yaml]
skills: []
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/messages/... ./internal/admin/messages/... && make test-int PKG="./internal/messages/... ./internal/admin/messages/..." && make contract ROUTES=messages
---

# T-M10-03 — Message fact registry v6 with lazy producers
**Model:** opus (facts drive health messages).

## Why
SPEC §5.3 lists the fields a rule may test. Thirteen of them do not exist (`days_late`, `confidence`,
`data_quality`, `warnings`, `requires_user_input`, `cycle_regularity`, `suggest_pregnancy_test`,
`hormonal_contraception`, postpartum week, last vitals category, `symptom_history`, account facts, age), fact names are
free text and never validated (`validate.go:155-158`), and every request builds every fact (gap 03 §2.3, §2.9).
T-M9-38 put the v6 engine facts into the old builder; this task turns facts into a typed registry with lazy,
grouped producers so a home render costs one engine state load plus at most a few small queries. DECISIONS.md M-06,
M-14; D-161.

## Scope
1. Package `internal/messages/facts`:
   - `Def{Name, Type (int|float|bool|enum|set|date), Enum []string, Unit, Group, Description{fa,en}, MemberSafe bool}`
     registered per group file (`engine.go`, `account.go`, `logs.go`, `vitals.go`, `pregnancy.go`, `postpartum.go`,
     `contraception.go`) through `Register(...)` — module tasks add their own files later (T-M10-08, T-M10-09)
     without editing a central list.
   - `Facts` per request: lazy and memoised per producer group; `Need(names...)` prefetches the union in one pass
     (one engine state load for every engine fact; one query per other group).
2. Facts:
   - Engine (from the T-M9-32 state provider, target day = `LocalToday`): `mode`, `life_mode`, `main_phase`,
     `subphase`, `cycle_day`, `days_to_period`, `days_late`, `confidence`, `data_quality`, `warnings` (set),
     `requires_user_input`, `cycle_regularity`, `suggest_pregnancy_test`, `fertility_suppressed`, `is_fertile_window`
     (display window), `is_pms_window`, `today_ui.banner` (M-14), `current_period_start` (date, for cycle caps).
   - `hormonal_contraception` (I-03 rule, contraception module).
   - `pregnancy_week`, `pregnancy_trimester` (the pregnancy dating code — the same function the pregnancy screens use;
     T-M10-18 fixes the trimester boundary there), `postpartum_week`.
   - `bp_last_category`, `glucose_last_category` (last reading in 30 days).
   - `symptom_history.<code>` (number of cycles in which that symptom was logged in the same phase as today; pattern
     fact), `days_since_last_log`, `logs_30d`.
   - `plan` (free|trial|plus), `days_since_signup`, `age` (null when unknown), `user_goal`, `locale`, `segment` (set of
     saved segment ids; empty until T-M10-32, marked so in the registry), `push_opt_in` (false; M-09).
   - `fertility_facts` constant moves here: the facts that force the `fertile_window` topic (M-06).
3. Producers read through small read-only interfaces of the modules; `facts` never imports `admin` or `http`.
4. Validation: an unknown fact → 422 `unknown_fact`; operator/value type mismatch (`between` only on numbers/dates,
   `in`/`not_in` on enums/sets, `contains` on sets) → 422 `bad_value`; enum values checked against the registry.
5. `GET /api/admin/v1/messages/facts` returns the registry (types, enums, units, descriptions, member-safe flag) for
   the editor (T-M10-12).
6. `v2.go`: v6/shadow mode builds facts through the registry; v1 mode unchanged.

## Tests
- Table test per producer group with fixtures (engine personas: late 9 days, pill user, postpartum week 5, TTC,
  pregnant 30+2, teen).
- Laziness: with a counting fake, a request whose rules reference only engine facts runs zero pregnancy, vitals and log
  queries.
- `requires_user_input = true` and `warnings` present for an ambiguous-period persona; pill → `hormonal_contraception`
  true, copper IUD → false.
- `symptom_history.cramps` = 3 over three cycles with cramps in the menstrual phase.
- Validation: unknown fact 422, `between` on an enum 422, bad enum value 422.
- Registry endpoint golden (stable order).

## Budgets
- All engine facts from a warm state ≤ 50 µs; a cold full fact build ≤ 4 queries beyond the engine state.

## Out of scope
- Selection (T-M10-04), pregnancy/menopause/condition module facts (T-M10-08, T-M10-09), member fact sets (M11).

## Acceptance
- Verify passes; PROGRESS.md cites D-161.
