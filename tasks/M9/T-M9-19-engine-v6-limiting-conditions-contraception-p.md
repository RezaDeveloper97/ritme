---
id: T-M9-19
title: Engine v6 limiting conditions: contraception, pregnancy, postpartum, positive test
milestone: M9
type: backend
status: done
depends_on: [T-M9-18]
parallel_group: M9-B
touches: [backend-go/internal/cycle/engine]
skills: []
verify: cd backend-go && go vet ./internal/cycle/engine/... && go test -race ./internal/cycle/engine/...
---

# T-M9-19 — Engine v6 limiting conditions: contraception, pregnancy, postpartum, positive test
**Model:** opus (health logic, §2.40 forbidden rules).

## Why
None of §2.42 exists in the resolver: under the pill the fertile window and fertility level are shown (a §2.40
violation), pregnancy is handled outside the engine, postpartum predicts as if cycles were regular, and a positive
test changes nothing (gap 01 §2.42).

## Scope
1. Hormonal contraception (E-13): logged bleeding → `menstrual`; from `predicted_next` while no bleed is logged →
   period_expected tiers (no `suggest_pregnancy_test`); every other day → main/subphase/fertility `unknown`; ovulation
   fields and `days_to_ovulation` null; `ovulation_source = unknown`; warning + reason `hormonal_contraception_active`;
   confidence low; `requires_user_input` false; `limits.fertility_suppressed = true`, reason
   `hormonal_contraception`. These "unknown" phases never add `insufficient_anchor_data`.
2. Pregnancy mode (E-25): suspended status — `main_phase`/resolution/confidence `unknown`, no predictions or ovulation,
   `limits.engine_suspended = true`, reason `pregnancy`, `requires_user_input` false; real logged periods in the past
   still resolve as `menstrual` for history views.
3. Postpartum or breastfeeding (E-14): until 2 real periods have started after `DeliveryDate`, logged period days →
   menstrual; other days → `unknown`, no `predicted_next`, no ovulation, `days_to_*` null, warning
   `postpartum_cycles_not_returned`, `fertility_suppressed` (reason `postpartum_not_returned`); confidence `unknown`
   with no postpartum period and `low` after the first (documented precedence exception). Breastfeeding without a
   delivery date applies the rule from the first real period on record (documented).
4. Positive pregnancy test (E-26): a positive test on/after `current_period_start` in mode cycle/ttc →
   `limits.prompt_pregnancy_mode = true`; cleared by a new real start or a mode change.
5. Menopause and teen modes (E-29): computed normally; no suppression.
6. §2.26/§2.34: `fertility_level = unknown` and `days_to_ovulation = null` whenever fertility is suppressed;
   `suggest_pregnancy_test` false under hormonal contraception and in pregnancy/postpartum.
7. Export `State.Suppression(today)` so T-M9-20 (future cycles) and consumers read one flag.

## Tests
- §6.1 pill: 28-day user with a combined pill → cycle day 14 `unknown/unknown/unknown`, no window, `days_to_ovulation`
  null, warning + reason, low, `fertility_suppressed`; a logged withdrawal bleed → menstrual; 3 days after
  `predicted_next` without a bleed → period_expected, `suggest_pregnancy_test` false.
- Copper IUD (`Hormonal = false`) → normal output.
- Pregnancy mode → suspended status for today; a period logged before the pregnancy still resolves menstrual on its
  date.
- §6.1 postpartum without a period: delivery 2026-08-01, breastfeeding, no period → `unknown`, warning, suppressed,
  confidence unknown; one period on 2026-10-10 → that period menstrual, other days unknown, confidence low; second period
  on 2026-11-08 → normal prediction resumes (from the second start).
- Positive test 2026-07-20 after start 2026-07-01 in cycle mode → prompt; next real start 2026-08-02 → no prompt; in
  pregnancy mode → no prompt.
- Invariant (table over all fixtures): fertility suppressed ⇒ no day has main_phase `fertile` and fertility_level is
  `unknown`.

## Out of scope
- Mapping contraception methods to `Hormonal` (T-M9-25), future cycles under suppression (T-M9-20 uses the flag).

## Acceptance
- Tests green; §2.40 rule 9 ("no fertile window or fertility level under hormonal contraception") holds in every test.
