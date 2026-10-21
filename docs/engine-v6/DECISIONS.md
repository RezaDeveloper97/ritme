# Engine v6 — open decisions

Every place where `SPEC.md` is silent, ambiguous or contradicts itself, or where the code needs a product or clinical
choice, is listed here with a **recommended default**. The tasks implement the default unless the decision is changed
before they run.

- **Owner**: `user` = product owner (answers in chat), `advisor` = medical advisor (clinical sign-off), `user+advisor`
  = both.
- **Status**: `proposed` → `accepted` (owner said OK) or `changed` (owner picked something else — write what) or
  `advisor-pending` (shipped with the safe default below, waiting for the advisor).
- **Who records the answers**: T-M9-01 (sections E, I, X), T-M10-01 (M, H, A), T-M11-01 (F). Nothing is left
  `proposed` once that task is done.
- **Approval:** sections E, I and X accepted by the user on **2026-10-21** (T-M9-01) — every recommended default,
  no changes.
- **Clinical rule of thumb**: a change that *adds* an escalation (more alerts, earlier urgency) ships with the spec
  value. A change that *removes or lowers* an escalation ships as a clinical-config value that keeps today's behaviour,
  so the advisor's sign-off is a config approval in admin (two advisors, audited), not a code change.
- Source reports with evidence: `gap/01…07`. Spec section numbers refer to `SPEC.md`.

---

## E — Cycle engine (decided in T-M9-01)

### E-01 — When is `prediction_based_output` emitted?
- **Question:** §2.35's worked example resolves via `prediction` but shows `warnings: []` and
  `confidence_reasons: []`; §2.32 lists `prediction_based_output`, and v1.1 emits it on every prediction-resolved day.
- **Recommended:** emit it only when the current anchor is `predicted_reference` (stale onboarding anchor §2.41, or a
  skipped cycle E-15) or the target is resolved from a `future_cycles` projection (E-03). Ordinary days resolved from the
  current cycle's prediction carry no warning, so the §2.35 example holds byte for byte.
- **Alternative:** keep v1.1 (always on `prediction`) and treat the example as illustrative.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-18

### E-02 — `contradictory_anchor_data`
- **Question:** §2.33 lists it as a `requires_user_input` trigger, but it is missing from the §2.32 warning enum and no
  rule produces it.
- **Recommended:** add it to the warning enum (21 values). Raise it for: `end < start`; two real periods that overlap;
  a real start inside another real period; more than one open period. It implies `data_quality = poor`,
  `confidence = low`, `requires_user_input = true`, and the engine ignores the newer of the two conflicting records for
  medians.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-15, T-M9-18

### E-03 — Resolving future dates
- **Question:** §2.29 makes a `predicted_reference` start **low**; §2.16 says future cycles k ≥ 2 are "at most medium",
  which implies medium is reachable for future dates.
- **Recommended:** `period_expected` (lateness) applies only to targets ≤ today. A target after today that lies on or
  after the current cycle's next start resolves inside `future_cycles[k]`: its start is `future_cycles[k].period_start`
  with source `predicted_reference` (the only honest §2.10 value), warning `prediction_based_output`, and confidence
  `min(today's confidence, k == 1 ? high : medium)`. The forced `low` of §2.29 for a `predicted_reference` start applies
  only to the stale onboarding anchor (§2.41) and skipped cycles (E-15), not to future-cycle projections.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-20

### E-04 — The future chain while the period is late
- **Question:** §2.16 chains from `predicted_next_period_start`, which is already in the past once `days_late > 0`.
  v1.1 rolls the anchor forward and shows "period day 5" tomorrow while today says "3 days late".
- **Recommended:** rebase. While no new start is logged and `today ≥ predicted_next`, `future_cycles[1].period_start =
  today + 1`, and k ≥ 2 chain from it by `effective_cycle_length`. Dates from `predicted_next` to today resolve as
  `period_expected` (late tiers). When `days_late > max_late` (unknown), `future_cycles` is empty. The calendar never
  paints a rolled-forward period while the user is late.
- **Alternative:** keep the stale chain and mark it overdue.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-20, T-M9-45

### E-05 — Shape of a `future_cycles` item
- **Question:** the §2.35 example shows only `{k:2, period_start, ovulation}` and omits k = 1; which cycle does an
  item's ovulation belong to?
- **Recommended:** always k = 1..6, each
  `{k, period_start, period_end, fertile_start, fertile_end, ovulation, ovulation_source, confidence}`.
  An item describes period k and the ovulation that precedes it: for k = 1 the ovulation is the current cycle's
  `estimated_ovulation_date` with its real source (bbt_confirmed / lh_positive / calendar); for k ≥ 2 it is
  `period_start[k] − 14` (`calendar`). `period_end = period_start + effective_period_length − 1`; fertile window =
  `ovulation − 5 .. ovulation`. Items are never stored (§2.40).
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-20, T-M9-45

### E-06 — Basis of `clinical_flags.frequency`
- **Question:** §2.27 classifies `effective_cycle_length`, but §2.8 drops cycles < 21 and > 45 from it, so a woman with
  real 18-day cycles falls back to 28 and reads "normal".
- **Recommended:** classify the median of the last ≤ 3 *real complete* cycle lengths (outliers included, skipped cycles
  excluded). With no real cycle, use the declared length; with neither, `frequency = null`.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-18

### E-07 — `duration` and `absent`
- **Recommended:** `duration = prolonged` when the median of the last ≤ 3 *closed real* periods is > the FIGO maximum
  (`cycle.figo.duration`, 8); `null` with no closed real period. `absent = true` when at least one real period exists,
  `difference_in_days(today, last real start) ≥ 90`, no period is open and the mode is not pregnancy/postpartum;
  `null` with no real period (a declared onboarding date alone is handled by E-15/§2.41, not by `absent`).
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-18

### E-08 — Teen and over-45 thresholds
- **Question:** "teen" needs a menarche date the app does not collect; regularity thresholds for < 18 and > 45 are
  undefined.
- **Recommended:** teen = age < 18 (birth date) until menarche is collected (see I-06). Teens use the ACOG 21–45 range
  for `frequency` (frequent < 21, infrequent > 45). Regularity for < 18 uses the `young` band (≤ 9 days) and for > 45
  the `late` band (≤ 9 days).
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-18

### E-09 — Regularity window and outliers
- **Recommended:** `cycle_regularity` uses real complete cycles whose start lies in the 12 months before the target,
  at least 3 of them; skipped cycles are excluded; outliers (< 21, > 45) are included (they are real variation). The
  analysis tab's one-outlier exclusion (`analysis/cycle.go` `markOutlier`) is removed so the tab and `cycle_view`
  always agree.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-18, T-M9-35

### E-10 — Onboarding override of `resolution_source`
- **Question:** §2.30 says the enum order is "only a tie-breaker", which would put `prediction` above
  `onboarding_based` and make `onboarding_based` almost unreachable.
- **Recommended:** keep v1.1's override: when the anchor is the declared onboarding date, every phase resolves as
  `onboarding_based` (or `default_based` when both lengths are defaults), period_expected days 0–14 included. The
  tie-breaker applies only between genuinely equal sources.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-16

### E-11 — `resolution_source` after 14 days late
- **Recommended:** `prediction` (confidence therefore `low`, not `unknown`), matching "unknown main_phase is not an
  invalid output" in §2.22.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-16

### E-12 — Cap warnings are cumulative
- **Recommended:** an open period past the hard cap carries `period_end_missing`,
  `period_end_missing_warning_cap_exceeded` and `period_end_missing_hard_cap_exceeded` together, so every consumer keyed
  on any of them keeps working.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-16

### E-13 — Phases under hormonal contraception
- **Question:** §2.42 stops fertile and ovulation, which leaves follicular/luteal undefined.
- **Recommended:** logged bleeding days → `menstrual` (the UI calls it «خون‌ریزی قطع مصرف»); days from
  `predicted_next` while no bleed is logged → `period_expected` tiers (no pregnancy-test suggestion, §2.22); every other
  day → `main_phase = unknown`, `subphase = unknown`, `fertility_level = unknown`; ovulation fields and
  `days_to_ovulation` null; `ovulation_source = unknown`; `future_cycles` keep period dates only (fertile/ovulation null);
  `limits.fertility_suppressed = true` (E-24); warning + reason `hormonal_contraception_active`; confidence `low`;
  `requires_user_input = false`. "unknown" here never means "insufficient data".
- **Alternative:** `follicular / mid_follicular` for all other days (a placeholder phase the UI must then hide).
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-19, T-M9-44

### E-14 — Postpartum and breastfeeding before cycles return
- **Recommended:** applies in postpartum mode, or in any mode while `breastfeeding = true`, until 2 real periods have
  started after the delivery date. Logged period days resolve as `menstrual`. Every other day: `main_phase = unknown`,
  no `predicted_next`, no ovulation, `future_cycles = []`, `days_to_*` null, warning `postpartum_cycles_not_returned`,
  `limits.fertility_suppressed = true` (reason `postpartum_not_returned`), `requires_user_input = false`. Confidence is
  `unknown` while no postpartum period exists and `low` after the first one (§2.29 lists "postpartum with < 2 real
  periods" as low — documented precedence exception).
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-19

### E-15 — Skipped cycle («پریود نداشتم»)
- **Recommended:** allowed when `today ≥ predicted_next` for the cycle being skipped (any `days_late ≥ 0`, including
  > 14). The skip stores the predicted start it refers to. The engine then uses that date as a virtual anchor:
  `current_period_start = skipped date`, source `predicted_reference` (so confidence `low`), cycle_day restarts there,
  `predicted_next = skipped date + effective_cycle_length` (§2.39.8). The gap between the real start before the skip and
  the next real start is neither a valid record nor an outlier; it is excluded from medians, variability and regularity.
  `skipped_cycle_recorded` stays until the next real start. Consecutive skips are allowed. A skip can be deleted, which
  restores the previous output.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-20, T-M9-29, T-M9-47

### E-16 — BBT and LH rule variants
- **Recommended:** BBT strict mode by default (`engine.bbt.strict = 1`: the third high reading must be ≥ +0.20 °C over
  the max of the previous 6); "consecutive" means consecutive *readings* (a missing day does not break the run);
  ≥ 10 readings in the cycle; temperatures handled as integer hundredths of °C. LH: only `positive` counts (`faint` is
  ignored). LH/BBT conflict threshold 2 days (`> 2` → BBT wins + `lh_bbt_conflict`). `effective_luteal_length` = the §2.9
  median (last ≤ 3, half up) of BBT-confirmed luteal lengths (next real start − BBT ovulation) that lie in 10..16;
  otherwise 14. LH-only cycles do not contribute luteal lengths (spec: "BBT-confirmed").
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-17

### E-17 — Who owns the §2.43 Today mapping
- **Recommended:** the engine emits a pure `today_ui` block, so web, the daily card and any later client cannot drift:
  `{ring_action: start|end|edit, banner: null|end_missing|overdue|ask_last_period, show_pregnancy_test_button,
  prediction_card: {period:{start,end}, pms:{start,end}, fertile:{start,end}|null, ovulation:{date,source}|null,
  cycle_length, cycle_regularity}}`. «هنوز نه» / «هنوز ادامه دارد» stay client-local for the day (spec).
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-21, T-M9-42

### E-18 — Periods of 9–10 days
- **Question:** §2.8 says they "may lower confidence depending on history"; no rule is given.
- **Recommended:** no rule in v6.0 (documented). They stay valid records.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-18

### E-19 — Making the BBT path to `high` reachable
- **Question:** §2.29 checks `low` (which includes `cycle_variability > 5`) before `high`, so "variability ≤ 5 **or**
  ovulation_source = bbt_confirmed" can never reach high through BBT.
- **Recommended:** the low condition reads "cycle_variability > 5 **and** ovulation_source ≠ bbt_confirmed".
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-18

### E-20 — Manual cycle/period lengths (B-N1-09)
- **Recommended:** keep the user's "use my lengths" switch as an explicit override. The engine treats the values as
  declared: `data_quality ≤ partial`, never `high`, reason `onboarding_data_used` is *not* added (it is not onboarding);
  `cycle_length_source = declared`.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-15, T-M9-18

### E-21 — An open period past the warning cap
- **Question:** spec-faithful output keeps the whole cycle `poor / low / requires_user_input` until the user closes the
  period.
- **Recommended:** keep it (spec-faithful). It is softened by the end prompt («پریودم تموم شد», shown from
  `effective_period_length − 1`, §2.43), the `end_missing` banner, and the bridge closing an open period at its last
  logged bleeding day when the next period starts (I-02).
- **Alternative:** limit it to days ≤ hard cap + N.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-16

### E-22 — Layout of completed past cycles
- **Question:** the spec defines resolution around one current cycle. For a past target inside a *completed* cycle
  (the next real start is known), should the engine still predict from `start + L` (showing "late" days inside history)?
- **Recommended:** retrospective layout. Inside a completed cycle the actual next start replaces the prediction:
  ovulation = BBT/LH if detected, else `next real start − 14`; fertile/luteal/PMS are laid out over the real length;
  no `period_expected` days exist inside a completed cycle. `resolution_source = prediction` (phases are still
  estimates). This keeps the calendar, the analysis tab and fertility history consistent.
- **Alternative:** spec-literal (predict every past day from its own cycle start).
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-20, T-M9-33

### E-23 — Reading of `data_quality = good`
- **Recommended:** keep v1.1's stricter reading: `good` also needs the period-length source to be real history.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-18

### E-24 — A `limits` block in the output
- **Question:** clients need one clear flag instead of inferring suppression from warnings (02c question 3).
- **Recommended:** additive block
  `limits: {fertility_suppressed: bool, suppression_reason: null|hormonal_contraception|pregnancy|postpartum_not_returned,
  engine_suspended: bool, prompt_pregnancy_mode: bool}`. Also copied into the companion payload as
  `fertility_suppressed` only.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-19, T-M9-21, T-M9-32, T-M9-44

### E-25 — Output in pregnancy mode
- **Recommended:** the cycle engine is suspended (§2.42): `cycle_view` returns `main_phase = unknown`,
  `resolution_source = unknown`, `confidence = unknown`, no predictions, no `future_cycles`,
  `limits.engine_suspended = true`, `suppression_reason = pregnancy`, `requires_user_input = false`. `/cycle/month`
  still shows logged periods (history) with no predicted days.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-19, T-M9-33

### E-26 — Positive pregnancy test
- **Recommended:** a positive test dated on or after `current_period_start`, in mode cycle or ttc, sets
  `limits.prompt_pregnancy_mode = true` (UI: «رفتن به حالت بارداری»). It clears when a new period starts or the mode
  changes. It does not change phases by itself.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-19, T-M9-44

### E-27 — Age rules
- **Recommended:** age is computed at the target date from `birthday`. No birthday → no age rules. ≥ 45 → confidence
  capped at medium + `perimenopause_age_range`. < 18 → teen flags (E-08). Teen users with only an age band
  (`teen_profiles.age_band`) are treated as < 18.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-18, T-M9-19

### E-28 — Targets beyond the 6th future cycle
- **Recommended:** `main_phase = unknown`, `resolution_source = unknown`, no fertile data. The calendar shows nothing
  predicted beyond `future_cycles[6].period_end`.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-20

### E-29 — Menopause and teen modes
- **Question:** §2.42 does not mention the menopause and teen life modes; today they only hide fertility *copy*
  (`model.Profile` no-fertility flag).
- **Recommended:** the engine computes these users normally (age rules E-08/E-27 still apply);
  `limits.fertility_suppressed` stays false because it is reserved for clinical suppression. The existing presentation
  flag that hides fertility copy for teen and menopause users stays as it is in every consumer.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-19, T-M9-46

---

## I — Engine inputs and storage (decided in T-M9-01)

### I-01 — Onboarding seed row
- **Question:** onboarding writes the declared LMP into `cycle_histories` as a confirmed `user_profile_confirmed` row
  with an assumed end (`profile/service.go:463-513`), which breaks §2.2, §2.9, §2.40 and makes §2.41 unreachable.
- **Recommended:** new column `user_profiles.onboarding_last_period_start` holds the declared anchor; nothing else
  writes it. Data migration: an untouched seed (start/end equal what a re-sync would write) is deleted and its start
  moves to the column; a seed the user edited becomes a confirmed `user_logged` row; `down` restores the rows.
  `syncOnboardingPeriodLog` is removed. The calendar shows the declared onboarding period as an *assumed* range
  computed by the engine (styled as predicted), which reverses the August 2026 "paint it as logged" decision.
- **Alternative:** keep the row as `is_estimated = 1`, `source = onboarding_estimate`, no end, no length.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-24, T-M9-27, T-M9-45

### I-02 — Bleeding-log → period bridge (revises D-55)
- **Recommended:** periods created from `flow` logs are actual data: `source = bleeding_log`, `is_confirmed = 1`
  (the engine reports them as `user_logged`). Deleting a log, deleting a day or clearing `flow` rebuilds the
  `bleeding_log` period from the remaining bleeding days (shrink, split or delete). The bridge never edits rows created
  by the period button. Every bridge write recomputes the stored `cycle_length` of the neighbours. In effective mode
  pregnancy a `flow` log raises the bleeding alert only — no period row, no LMP change. Editing a row through
  `PUT /cycle/period/{id}` marks it confirmed `user_logged`. A data migration converts existing unconfirmed,
  non-estimated `user_logged` rows to `bleeding_log` confirmed. `PromoteLoggedStart` is removed. The end rule is
  unchanged: the newest bridge period stays open (assumed end, §2.11) until the user confirms an end (end prompt or
  period editor) or the next period starts, which closes it at its last logged bleeding day; absence of logs is never
  read as "bleeding stopped".
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-28

### I-03 — Which contraception counts as hormonal
- **Recommended:** hormonal = `combined_pill`, `progestin_pill`, `hormonal_iud`, `injection`, `implant` (active method in
  `contraception_methods`). Not hormonal: `copper_iud`, `condom`, `other`. Onboarding `medications`:
  `contraceptive_pill` → hormonal; `iud` (type unknown) → not hormonal + a one-time prompt to pick copper/hormonal in
  contraception setup; `hormonal_medication` → not contraception, ignored by the engine.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-19, T-M9-25

### I-04 — Breastfeeding source
- **Recommended:** explicit nullable flag `postpartum_profiles.breastfeeding`, asked on the postpartum screens; not
  derived from `baby_feeds`.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-24, T-M9-25, T-M9-28

### I-05 — How postpartum users log returning periods
- **Recommended:** the log sheet offers `flow` in postpartum mode from delivery + 42 days (earlier bleeding is lochia);
  the bridge then creates periods as in cycle mode. The 42-day value is a clinical-config key
  (`postpartum.flow_from_day`, advisor may change it).
- **Owner:** user+advisor · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-28

### I-06 — Teen menarche
- **Recommended:** do not ask for the menarche year now; apply only "age < 18" (E-08).
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-18

### I-07 — Cervical mucus
- **Recommended:** stays out of the engine (no §2.16–§2.34 rule reads it); only the TTC analysis uses it.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-25

### I-08 — One "normal" criterion
- **Recommended:** `/home` health-check items and every UI text use `clinical_flags` / `cycle_regularity` (FIGO). The
  21–35 / 2–8 constants in `home/sections.go:922-1036` are removed and 21–45 is never called "normal" (§2.40).
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-34, T-M9-46

### I-09 — Duplicate `user_profiles` rows
- **Recommended:** count first (read-only; ask the user before querying beta). If duplicates exist: keep the lowest id,
  fill its NULL columns from the newest duplicate, copy every removed row into `user_profiles_dedupe_backup`, then add
  `UNIQUE(user_id)`.
- **Alternative:** fail the migration loudly and fix by hand.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-23

### I-10 — Derived cycle values stored outside the engine
- **Recommended:** the shop supply reminder is re-synced on every engine revision bump; the lab interpretation stores
  the phase computed for the lab's `taken_on` date (labelled with that date); `todo_suggestion_events.ref_date` stays
  as is (documented).
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-37

### I-11 — v6 phase → legacy content phase (daily messages, tips, v2 facts)
- **Question:** v6's fertile phase lasts 6 days; legacy "ovulation" content was written for 1–2 days.
- **Recommended:** menstrual → `menstruation`; follicular and `fertile_rising` → `follicular`; `fertile_high` and
  `ovulation_likely` → `ovulation`; luteal and period_expected → `luteal`; `is_pms_window` = `pms_possible` days
  (next − 3 .. next − 1); `unknown` → no phase content (neutral tips only).
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-38

---

## X — Cross-cutting (decided in T-M9-01)

### X-01 — Build a new engine package
- **Recommended:** new pure package `internal/cycle/engine` (no I/O, no globals, no clock, `Params` passed in, epoch-day
  integers inside, hand-written v6 enums). The v1.1 `resolver`/`view`/`legacy` stay frozen behind the rollout flag until
  T-M9-51 deletes them.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-15

### X-02 — Cache design and messaging infrastructure
- **Recommended:** DB-authoritative per-user revision (`user_revisions.engine_rev`, bumped inside every engine-input
  write transaction, read for free in the auth query) → L1 in-process LRU of the prepared engine state + L2 Redis input
  snapshot + singleflight; ETag/304 on engine GETs. Redis + asynq stay the only infrastructure; **no RabbitMQ, NATS or
  Kafka** (one server, one API replica per environment, tens of jobs per minute — README §6).
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-24, T-M9-30, T-M9-31

### X-03 — Rollout
- **Recommended:** env flag `CYCLE_ENGINE = v1 | shadow | v6` (default `v1`). Stage: `shadow` (serve v1, compute v6,
  record classified diffs) → `v6` after review. Beta: `v6` only after the user's OK, with a DB backup first. Production
  stays on Laravel (T-M2-26/27 untouched).
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-32, T-M9-48, T-M9-50

### X-04 — Goldens
- **Recommended:** the 124 Laravel-recorded goldens with `cycle_view`/`main_phase` stay as v1.1 characterization and run
  with `CYCLE_ENGINE=v1` until T-M9-51. New Go-native v6 goldens over the contract personas, regenerated with `-update`,
  with anchors hand-checked against the §2.36 tables and the §2.35 example.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-22, T-M9-32, T-M9-51

### X-05 — Legacy payload keys
- **Recommended:** keep `phase`, `data_status`, `predictions`, `profile_values`, `calculated_values`,
  `effective_values`, `data_quality_details`, `daily_card` and the per-day `calculation` block during M9, rebuilt from
  the v6 status (`predictions` window O−5..O, one confidence everywhere). Remove them from the Go output in T-M9-51 once
  the frontend reads only v6 fields (the frontend keeps parsing them from Laravel on production).
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-32, T-M9-51

### X-06 — One definition per concept (spec, informational)
- Fertile window = O−5..O (display window §2.19); PMS = next − 3 .. next − 1; late = §2.22 tiers; regularity = §2.27;
  "today" = user-local day (X-07). Every consumer reads the engine; no module keeps its own copy. Recorded so reviewers
  can reject a new private copy.
- **Owner:** — · **Status:** accepted (spec)

### X-07 — User-local day (§2.3)
- **Recommended:** the web client sends `X-Timezone: <IANA>`; the API validates it, falls back to the stored
  `user_profiles.timezone`, then to `Asia/Tehran`, and stores a changed zone (at most once per day). `LocalToday(c)`
  replaces `civildate.Today` on the engine path and its direct consumers in M9; the other call sites stay on Tehran and
  are listed in the task. DB timestamps stay Tehran wall-clock.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-24, T-M9-26

### X-08 — Deviation ids
- To avoid collisions with the bloom/canvas queues, Engine v6 uses reserved blocks in
  `docs/go-migration/deviations.md`: **D-120…D-159 (M9)**, **D-160…D-189 (M10)**, **D-190…D-209 (M11)**. The decisions
  task of each milestone writes the rows once the owner has answered; implementing tasks reference the id in contract
  allow-lists and never edit `deviations.md` themselves (differences go to `tasks/PROGRESS.md`; the milestone's release
  task reconciles the rows).

| Id | Change | Task |
|---|---|---|
| D-120 | `cycle_view` v6 (semantics, new fields, `today_ui`, `limits`, legacy keys rebuilt) | T-M9-32 |
| D-121 | `/cycle/month` from the v6 range (no wrap, lateness, O+1 fix, future cycles, suppression) | T-M9-33 |
| D-122 | `/home`, `/home/cycle-overview`, daily card on v6; health check on FIGO flags | T-M9-34 |
| D-123 | Onboarding seed replaced by the declared anchor (+ data migration) | T-M9-27 |
| D-124 | Bleeding-log bridge v6 (revises D-55) + postpartum flow | T-M9-28 |
| D-125 | Skipped-cycle endpoint (Go-only route) | T-M9-29 |
| D-126 | User-local day (`X-Timezone`, stored zone) | T-M9-26 |
| D-127 | `user_profiles.user_id` UNIQUE + dedupe | T-M9-23 |
| D-128 | Fertility BBT/LH rule v6, per-cycle ovulation from the engine | T-M9-35 |
| D-129 | Analysis regularity from the engine (`insufficient_data`, no `markOutlier`) | T-M9-35 |
| D-130 | Messages daily/selected facts from the v6 engine | T-M9-38 |
| D-131 | Period writes serialised per user; duplicate start → idempotent 200 | T-M9-04 |
| D-132 | ETag/304 and gzip on engine GETs | T-M9-10, T-M9-31 |
| D-133 | Family privacy hotfix (mood, notes, lateness, postpartum ≠ pregnancy, grants reset per episode) | T-M9-02 |
| D-134 | Pregnancy emergency additions (kick counter, ≥160/110, vitals in pregnancy, GDM ≥) | T-M9-05 |
| D-135 | Postpartum red flags, visit reminders, EPDS urgent threshold as config | T-M9-06 |
| D-136 | Small consumers on the engine (companion cap, teen, checkups, settings, todo/shop gating, labs date, conditions/search bounds) | T-M9-36, T-M9-37 |
| D-137 | Engine daily snapshots + prediction log (Go-only analytics tables) | T-M9-39 |
| D-138 | `bleeding_log` data source is Go-only (add it to Laravel `DataSource.php` before any cutover) | T-M9-28 |
| D-139 | v1.1 engine retired; Laravel cycle goldens no longer replayed | T-M9-51 |

### X-09 — Performance budgets
- **Recommended:** SLOs and benchmark ceilings in README §8 are the initial targets. T-M9-08 records the real baseline,
  then locks benchmark ceilings at baseline × 1.2. CI fails on an `allocs/op` regression and warns on a `sec/op`
  regression > 20 % (shared CI runners are too noisy to fail on time).
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-08, T-M9-49

### X-10 — Worker process
- **Recommended:** the same binary gains `api worker` (asynq server + scheduler + pollers); stage and beta run it as a
  second container; the API container sets `API_RUN_WORKERS=0`. Production compose is not touched.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-12

### X-11 — Which engine numbers are clinical config
- **Recommended:** clinically tunable values live in `clinicalconfig` (two-advisor approval): caps and late days
  (existing), fertile `before` (existing; `after` retired in T-M9-51), `engine.luteal {default 14, min 10, max 16}`,
  `engine.bbt {strict 1, min_readings 10, rise_centi 20}`, `engine.lh_bbt_conflict {days 2}`,
  `engine.pregnancy_test {cycle_days_late 8, ttc_days_late 1}`, `engine.stale_anchor {extra_days 14}`,
  `cycle.figo.*` (existing), `cycle.teen.frequency {min 21, max 45}` (ACOG CO 651). Pure product constants
  (6 future cycles, banner order, prompt window −3..+14) are code constants in `engine.DefaultParams()`.
- **Owner:** user · **Status:** accepted (user, 2026-10-21) · **Blocks:** T-M9-15

---

## M — Message engine (decided in T-M10-01)

| Id | Question | Recommended default | Owner · Blocks |
|---|---|---|---|
| M-01 | Rename DB fields to the spec names (`type`, `slot`, `shown`…)? | Keep the columns (`kind`, `placement`, `impression`); accept the spec names as API aliases; the admin UI shows spec names. | user · T-M10-02 |
| M-02 | Does a non-text edit (priority, audience, schedule, caps) void clinical approval? | Only content, conditions, level or kind changes void it. The version still bumps. | user · T-M10-02 |
| M-03 | While a new version awaits approval, show the old one or nothing? | Serve the last `approved_version` snapshot (no gap for users). | user · T-M10-02 |
| M-04 | `max_per_cycle` = real cycle or rolling window? | The real cycle from the engine (`current_period_start`); rolling `effective_cycle_length` only when there is no anchor. | user · T-M10-04 |
| M-05 | How does a clinical rule allow `confidence = low`? | Allowed iff the rule has a condition on the `confidence` fact. | user · T-M10-04 |
| M-06 | How is a "fertile-window message" identified (removed under hormonal contraception)? | A `topics` list on the message with `fertile_window`; the validator requires it when the conditions or text use fertile/ovulation facts. | user · T-M10-02, T-M10-04 |
| M-07 | Is safety exempt from audience and A/B, or only from caps? | Exempt from all three; the validator forces audience 100 and a single variant. | user · T-M10-02 |
| M-08 | Timezone for schedules and quiet hours | The user-local day and clock from X-07 (supersedes "Tehran everywhere"). | user · T-M10-04 |
| M-09 | Push channel | Model the push slot, caps and quiet hours (23–08 local) in the engine; deliver to the in-app `user_notifications` inbox; real web push is a later milestone. | user · T-M10-04, T-M10-07 |
| M-10 | Pregnancy alert detectors | Become fact/evidence producers; thresholds come from the clinical registry; seeded rules reproduce today's behaviour; `pregnancy_alerts` rows move into the inbox. | user · T-M10-08 |
| M-11 | What a member sees of an owner's urgent alert | A fixed neutral text per level, the time, "call her" and 115 — never the rule title, body or evidence (same as F-10). | user · T-M11-12 |
| M-12 | Logging `eligible` | Log `eligible` for the evaluated top-K (≤ 5) of each slot, sampled 10 % for product/marketing; always log shown, clicked, dismissed, goal. | user · T-M10-06 |
| M-13 | Retention of `message_events` | Raw rows 180 days; daily rollup `(rule, version, variant, slot, event, day, count)` kept forever. | user · T-M10-06 |
| M-14 | Who owns the Today status banner: the engine's `today_ui.banner` or the message engine's `status_banner` slot? | The engine decides *which* banner (E-17). The message engine supplies its copy and CTA through seeded `engine`-kind rules keyed on `today_ui.banner`; the client falls back to built-in copy when the feed is unavailable. | user · T-M10-06, T-M10-14 |
| M-15 | Fate of `/messages/daily` | Keep it for the legacy daily card until the home reads the feed (T-M10-14); then it is deprecated (still served for Laravel parity). | user · T-M10-06, T-M10-14 |
| M-16 | An inbox item that was acknowledged fires again | One open row per (user, message): count + 1 and `last_at` move; an `urgent` item goes back to `active` (it must be acknowledged again), other levels stay acknowledged. A resolved or archived item is history; the next hit opens a new row. | user · T-M10-07 |
| M-17 | `goal_event` attribution | Server-side: a goal counts when the named domain event (fixed catalogue, e.g. `period_logged`, `log_saved`, `checkup_done`, `plus_purchased`, `challenge_joined`) happens within 7 days after the message was last shown to that user. Client-posted `goal` stays accepted for client-only goals. | user · T-M10-06 |
| M-18 | Existing rules whose kind is not allowed in their slot (spec §5.4 slot table: smart tip = clinical, today's advice = content, challenge = product, modal = marketing) | The migration re-maps the unambiguous cases (`daily_advice` + product → content, `modal` + product → marketing); every other mismatch stays as is, is skipped by selection with reason `kind_not_allowed`, and is listed for the admin. | user · T-M10-02 |
| M-19 | How often `shown` is logged | Once per (user, message, slot, user-local day): refreshing the home does not count as another impression, so caps count days, not fetches. | user · T-M10-06 |

---

## H — Health modules (decided in T-M10-01; clinical items need the advisor)

| Id | Question | Safe default shipped now | Spec / recommended | Owner · Blocks |
|---|---|---|---|---|
| H-01 | EPDS total ≥ 13 without Q10: open the 123/115 crisis page? | Unchanged: config `epds.urgent_total = 13` (today's behaviour). | Not urgent: "high probability, talk to your doctor soon"; only Q10 > 0 is urgent. Advisor sets the key to 31 (off). | advisor · T-M9-06 |
| H-02 | Pregnancy BP ≥ 140/90 level | Unchanged: config `pregnancy.bp.high.urgent = 1` (1 = urgent, 0 = follow_up). ≥ 160/110 → urgent is added now. | ≥ 140/90 → follow_up ("call your doctor today"); ≥ 160/110 → urgent (115). | advisor · T-M9-05 |
| H-03 | Do vitals readings in pregnancy use the pregnancy table? | Yes (adds escalation). | Yes (§3.1 replaces §3.2 in pregnancy). | user · T-M9-05 |
| H-04 | Fetal movement start week | Kick-counter auto alert from week 28 (added now); manual reduced/none keeps week 24 via `pregnancy.fetal_movement.manual_from_week = 24`. | 28 for both; advisor decides whether "none" at 24–27 stays a warning. | advisor · T-M9-05 |
| H-05 | Kick session < 10 in 2 h → urgent automatically? | Yes (adds escalation), at session end or when the 2 h window expires. | Yes. | user · T-M9-05 |
| H-06 | 5-1-1 contraction duration | Unchanged: `pregnancy.contractions_511.duration_s = 45`. | Spec 1 minute (60 s); 45 s is more sensitive — advisor confirms. | advisor · T-M9-05 |
| H-07 | Borderline lab category (±10 %) | Kept until sign-off: config `lab.borderline {enabled 1, margin 0.10}`. | Remove as a class; at most a visual hint under "out of range" (advisor sets `enabled 0`). | advisor · T-M10-20 |
| H-08 | OCR confidence threshold and confirmation | 0.9 (stricter than today's 0.7, so it ships now). | 0.9 from the registry; values below it are excluded from rules and AI until the user confirms that value. | user · T-M10-20 |
| H-09 | Glucose classes for glucometer post-meal, random and bedtime readings | Unchanged: config `glucose.contexts {spec 0}` (0 = today's post-meal bands 140/200 for all three). | `spec 1`: post-meal < 140 in range, otherwise «با پزشک در میان بگذار» (no 200 class); random and bedtime: no class below 200 mg/dL. | advisor · T-M10-19 |
| H-10 | Breast self-exam history | — | `breast_self_exam` becomes an informational «آگاهی از سینه» item with no due date; history read-only. | user+advisor · T-M10-21 |
| H-11 | Pap/HPV at 30–65 | — | One checkup type whose interval depends on the last test kind (cytology 36 months, HPV/co-test 60 months). | advisor · T-M10-21 |
| H-12 | Tabular thresholds (IOM bands, re-dating table, screening, vaccines) | — | List-valued registry keys (JSON rows validated by a per-key schema), same two-advisor approval. | user · T-M10-16 |
| H-13 | Consent history | — | Append-only `consent_events` + `user_consents` kept as the current-state projection; `scope_json`; signup terms + health-data processing as versioned consent codes. | user · T-M10-23 |
| H-14 | Postpartum `flow` from day 42 | 42 | See I-05. | advisor · T-M9-28 |
| H-15 | The 3-month milestone band (not in CDC 2022) | Kept, labelled as Ritme's own band (not CDC). | Keep or drop — advisor. | advisor · T-M10-22 |
| H-16 | Checkup types not in the spec (clinical breast exam, yearly blood test) | Kept unchanged. | Keep, make optional, or retire — advisor. | advisor · T-M10-21 |
| H-17 | Which vaccine schedule version a child follows | The version in force on the child's birth date. | Same; the advisor confirms catch-up handling stays manual. | advisor · T-M10-22 |
| H-18 | Existing users and the new mandatory signup consents (terms + health-data processing) | — | A one-time acceptance sheet on the next app open; the answer is recorded as a consent event; nothing is blocked server-side while it is pending. | user · T-M10-23 |

---

## A — Analysis, KPI and admin (decided in T-M10-01)

| Id | Question | Recommended default | Owner · Blocks |
|---|---|---|---|
| A-01 | Analysis regularity source | The engine's `cycle_regularity` is the verdict; FIGO bands are shown only as the reference band. | user · T-M9-35, T-M10-26 |
| A-02 | Correlation method | Spec rule for `strength` and readiness (≥ 30 paired days, ratio ≥ 2×); φ kept as an extra field. | user · T-M10-26 |
| A-03 | Free lab-AI quota | 1 per month (spec), admin-tunable. | user · T-M10-29 |
| A-04 | Plus assistant fair-use cap | 300 messages per month, config-driven, soft message at the cap. | user · T-M10-29 |
| A-05 | Prediction-log semantics | Log both the first prediction emitted after a start and the last one before the next start; the MAE KPI uses `first` with `basis_cycles ≥ 2`. | user · T-M9-39, T-M10-30 |
| A-06 | "Active user" for nightly snapshots | Any session, log or event in the last 30 days. | user · T-M9-39 |
| A-07 | OKR on-track rule | `progress ≥ elapsed − 10 points` (spec). | user · T-M10-30 |
| A-08 | Churn thresholds | Spec thresholds including "7 days without activity → high". | user · T-M10-32 |
| A-09 | User reference code | `RT-` + first 6 base32 chars of HMAC(user_id, server secret), stored in a column for search. | user · T-M10-32 |
| A-10 | Postpartum analysis hub | In M10: backend endpoint + the existing `screens/analysis-postpartum`, behind the module kill-switch. | user · T-M10-27 |
| A-11 | SPEC §4.3 lists "prediction source and confidence, BBT with confirmation, LH-based prediction" as Plus | The engine computes and returns the same output for everyone, and the free Today/calendar keep the plain confidence wording (honesty and safety). Plus gates only the detail surfaces: the "why this date" breakdown (source, basis cycles, confidence reasons), the BBT chart's coverline/confirmation view and the LH-based prediction explanation. | user · T-M10-29 |
| A-12 | Where the §4.1 analysis minimums live | Product constants in one `analysis.Params` struct (documented, tested), not clinical config. | user · T-M10-26 |

**Approval (T-M10-01, 2026-10-21):** every M row (M-01…M-19), every A row (A-01…A-12, including A-11 as written) and the user-owned H rows (H-03, H-05, H-08, H-12, H-13, H-18) are **accepted** by the user — recommended defaults, no changes. Advisor rows (H-01, H-02, H-04, H-06, H-07, H-09, H-10, H-11, H-14, H-15, H-16, H-17) are **advisor-pending**: the safe default ships; see `clinical-signoff.md` (also appended to the review doc «بازبینی پزشکی محتوای ریتمی»).

### M10 deviation ids (X-08 block D-160 … D-189)
T-M10-01 writes these rows in `docs/go-migration/deviations.md`; implementing tasks cite the id and never edit that file.

| Id | Change | Task |
|---|---|---|
| D-160 | Message model v6: the last approved version stays live during re-approval; only text/condition/level/kind edits void approval; `content` and `marketing` kinds; several caps per rule; N weighted variants; datetime schedule with several local hour windows; safety and clinical authoring need the clinical capability | T-M10-02 |
| D-161 | Message facts come from the v6 engine and the modules through one fact registry; unknown facts are rejected | T-M10-03 |
| D-162 | Selection v6: fixed invariants, caps counted from shown or dismissed, real-cycle caps, least-recently-shown tie-break, per-slot kinds and capacities, user-local schedule and quiet hours | T-M10-04 |
| D-163 | `GET /messages/feed` (Go-only); server-logged `shown`/`eligible`; spec event names accepted; server-matched goals; `/messages/selected` wraps the feed | T-M10-06 |
| D-164 | Alerts inbox `/api/v1/alerts` with its lifecycle; the push slot is delivered to the in-app notifications inbox | T-M10-07 |
| D-165 | Pregnancy alerts come from the message engine; the v1 pregnancy alert engine is retired; `/pregnancy/v2/alerts*` read the inbox (supersedes D-21(c)) | T-M10-08 |
| D-166 | Menopause messages and condition nudges are rules (the old GETs are adapters) | T-M10-09 |
| D-167 | Challenges: eligibility and contraindications as rules, `on_phase_start`, suggestion through the daily_challenge slot | T-M10-10 |
| D-168 | Home reads the feed; `/messages/daily` deprecated but still served | T-M10-14 |
| D-169 | Pregnancy dating: trimester by completed days, ACOG re-dating, optional cycle-length correction | T-M10-18 |
| D-170 | Vitals: glucometer post-meal and random/bedtime classes, diabetic targets with a doctor override, report minimums, thresholds interpolated into copy | T-M10-19 |
| D-171 | Labs: per-value OCR confirmation at 0.9, mode-aware fallback ranges, HbA1c bands, borderline as config, JSON summary with a diagnostic-word filter | T-M10-20 |
| D-172 | Checkups: USPSTF intervals from the registry, breast awareness, follow-up result, "after pregnancy" label | T-M10-21 |
| D-173 | Children: percentile-line crossing, CDC 2022 bands, versioned vaccine schedule | T-M10-22 |
| D-174 | Consent ledger (`consent_events`, `scope_json`) and versioned signup consents | T-M10-23 |
| D-175 | Analysis methods per §4.1 (minimums, correlation strength, mood by phase, top finding, nudge in days, lab trend) | T-M10-26 |
| D-176 | Postpartum analysis hub (Go-only endpoint) | T-M10-27 |
| D-177 | Analysis cache keyed on `analysis_rev`; ETag/304 on analysis GETs | T-M10-28 |
| D-178 | Plus: Lab AI 1/month free, assistant fair-use cap, 1-year analysis range, detail surfaces per A-11 | T-M10-29 |
| D-179 | KPI definitions from engine snapshots and the prediction log; OKR status rule | T-M10-30 |
| D-180 | Automatic metric alerts | T-M10-31 |
| D-181 | Admin user monitoring: HMAC reference code, spec churn rule, segments, engine card | T-M10-32 |
| D-182 | Admin role matrix per §5.8; append-only, hash-chained audit log | T-M10-33 |
| D-183 | Safety self-test v2: persisted nightly runs over the live rule set | T-M10-24 |
| D-184 … D-189 | Reserved for differences found during M10 (recorded by T-M10-36) | — |

---

## F — Family (decided in T-M11-01)

| Id | Question | Recommended default |
|---|---|---|
| F-01 | What does "overall cycle phase" show a member? | `main_phase` only, plus an optional coarse "period soon" (≤ 3 days) for the partner role. No dates, cycle day/length, lateness or confidence. Friend role: phase only. |
| F-02 | Is the `fertile` phase label fertility data? Partner TTC switch? | In cycle mode the `fertile` label is allowed. In TTC mode the fertile window / ovulation / best days sit behind a partner-only switch (default off). Raw LH/BBT/intercourse logs are never shown. Under hormonal contraception `fertile` is never exposed. |
| F-03 | Drop the legacy `symptoms` section? | Yes. Existing grants become off; the owner gets a one-time "this sharing ended" card. (T-M9-02 already removes mood.) |
| F-04 | Meds: does "propose" need `write`? | Levels off/view/nudge/write where `write` = propose-only for the owner's records. Spouse default = nudge. |
| F-05 | Which member writes are proposals? | Only the owner's own health records (meds, appointments). Household to-do and carer child sub-scopes are direct writes, logged, with a 24 h undo. |
| F-06 | Nudge cap scope | 1 per dose/appointment overall; 3 per local day per (member, owner) pair. |
| F-07 | "Appointment not logged on time" | One nudge allowed between the owner's reminder time and the appointment time while it is still scheduled. |
| F-08 | Quiet hours for nudges | 23:00–08:00 owner-local is a hard floor, plus the owner's own quiet hours on delivery. Refuse with `quiet_hours`; never queue. |
| F-09 | Delivery channel for urgent member alerts | In-app inbox now; optional neutral SMS («پیامی از خانواده در ریتمی — اپ را باز کن») for urgent alerts only, per member, owner-controlled. |
| F-10 | Is the alert title a medical detail? | Yes. Members get a fixed neutral title per level, the time, "call her" and 115. |
| F-11 | Phone auto-bind and high-impact code invites | Bind to the new account as a pending acceptance (member sees the summary, accepts or declines). Phone invites live 7 days. Code-only invites that include child write, urgent alert or vitals need owner confirmation after acceptance. |
| F-12 | Expired/declined invites | Mark revoked (`revoked_by = expired|declined`), hide from lists, keep for audit/KPI. |
| F-13 | Carer time-box default | Pregnant: EDD + 42 days; postpartum: delivery + 42 days; otherwise 30 days; the owner can extend. |
| F-14 | Undo scope and payloads | Undo for member/guardian creates, edits and deletes on child and household data; sealed (AES-GCM) before/after JSON in the activity row, purged after 30 days; actor/action/time kept 12 months. |
| F-15 | Removing a parent | Removing the *other* guardian needs both; leaving yourself is unilateral; pending requests expire after 7 days; abuse cases go through support (audited). |
| F-16 | "Every allow is logged" vs read coalescing | Keep 15-minute read coalescing (documented deviation); writes, proposals, nudges and lifecycle events always logged. |
| F-17 | Existing link types | partner → partner; spouse → partner + co-guardian of shared children (`family_children` → guardians); teen `parent` stays a special role outside the v6 matrix. |
| F-18 | Plus for view-level members | SPEC §1.8 names a PDF report as the example. Recommended: when the owner has Plus, a view-level member gets the Plus analyses and a PDF summary of **only the domains they already see**; never the owner's full record PDF and never never-shared data. Optional task. |
| F-19 | Activity retention | Rows 12 months; sealed payloads 30 days. |
| F-20 | Domains a male owner may grant | meds, appointments, household (children through guardianship). |
| F-21 | 404 vs 403 for revoked access | 404 for unknown/revoked links (no existence oracle), 403 for an active link without the domain; tests assert "no data". |
| F-22 | New family screens have no artboard (Fam_Activity, PMeds, PCalendar, PAlert, PPostpartum, review card, approvals) | Build them now in the language of the existing Hamdam artboards (`docs/design/night-bloom/companion-family/`) and the design system; the main session reviews light/dark screenshots. Designs supplied later replace them. |
| F-23 | Which domains pause on an owner mode change? | Every owner-data domain (cycle phase, pregnancy, fetal activity, vitals, meds, appointments) pauses until the owner reviews (SPEC §1.8: the member sees only children and household meanwhile). The urgent-alert switch stays on (neutral content, safety). The review card offers one-tap "keep as before". |
| F-24 | "Night shift" on Fam_PPostpartum | A household to-do item «شیفت شب» assigned to one person for tonight (one tap); no new data type. |
| F-25 | Household list for a mutual pair | One household list per owner; a member sees each owner's list labelled with the owner's name; lists are never merged in storage. |
| F-26 | Child medication («دارو» in the child profile) | A dose log only (name, dose text, time, who gave it), written by guardians and visible to guardians; no schedule, reminders or dosing advice in M11. |

All F rows: **Owner** user · **Status** proposed · **Blocks** T-M11-02 … T-M11-23 (each task names the rows it reads).

### M11 deviation ids (X-08 block D-190 … D-209)
T-M11-01 writes these rows; implementing tasks cite the id and never edit `deviations.md`.

| Id | Change | Task |
|---|---|---|
| D-190 | Family schema v6 (link role/template/expiry, grants with mode epoch, owner mode state, activity columns) and the legacy-grant data migration | T-M11-02 |
| D-191 | Proposals, nudges, member alerts, member notices and family consent rows (schema) | T-M11-03 |
| D-192 | Child guardians, actor columns, child kept for the co-guardian on account deletion | T-M11-04, T-M11-11 |
| D-193 | `family.Authorizer` (check()) behind every member read and write; 404 for unknown/revoked links, 403 for a missing domain | T-M11-05 |
| D-194 | Member projections v6 (allow-list DTOs; symptoms section removed) | T-M11-06 |
| D-195 | Mode-change deactivation and owner review | T-M11-07 |
| D-196 | Invites v6 (preview, accept/decline, mutual request, phone auto-bind as pending, owner confirmation) | T-M11-08 |
| D-197 | Member writes to the owner's meds/appointments become proposals (202) | T-M11-09 |
| D-198 | Nudges | T-M11-10 |
| D-199 | Co-guardianship, undo, two-guardian approvals | T-M11-11 |
| D-200 | Member alerts (neutral relay of urgent alerts) | T-M11-12 |
| D-201 | Member home for every member, member tips, member notifications | T-M11-13 |
| D-202 | Household to-do domain and the child medication dose log | T-M11-14 |
| D-203 | Read-audit coalescing (15 minutes) kept | T-M11-05 |
| D-204 | Go-only `/api/v1/family/*`; `/companions/*` kept as a compatibility layer | T-M11-05 |
| D-205 | Admin family tab and family KPIs | T-M11-15 |
| D-206 | Plus for view-level members (optional) | T-M11-16 |
| D-207 … D-209 | Reserved for differences found during M11 (recorded by T-M11-23) | — |
