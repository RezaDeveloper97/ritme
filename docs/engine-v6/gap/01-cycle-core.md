# Gap analysis 01: core cycle engine vs SPEC v6 §2.3–§2.43

Scope: `backend-go/internal/cycle/{resolver,metrics,model,view}`, the `engine.*` keys in `internal/clinicalconfig`,
the cycle enums (`internal/enums`), and `internal/fertility/bbt` as the BBT reference. Read-only review, 2026-10-21.
Source of truth: `docs/engine-v6/SPEC.md` (Part 2 = lines 202–608, §6.1 tests = 902–918).
The known input/wiring gaps (snapshot inputs, onboarding row as `user_logged`, legacy `/cycle/month`, `/home`,
`calculation`, `view.Predict` O+1, cache key) come from the other report. They are only referenced here.

Legend: ✅ matches · ⚠️ differs (a concrete input gives a different output) · ❌ missing.

---

## 1. Summary table

| § | Rule | Status | One line |
|---|---|---|---|
| 2.3 | Gregorian dates, user's local day | ⚠️ | `civildate.Date` is Gregorian, and "today" is always Asia/Tehran (`civildate.go:59`) whatever the server TZ. The device timezone is never read. |
| 2.4 | difference_in_days | ✅ | `civildate.Date.DiffDays` (`civildate.go:105`). It rebuilds a `time.Time` on every call, which costs CPU. |
| 2.5 | cycle_day 1-based | ✅ | `resolver.go:134` |
| 2.6 | Inputs | ❌ | `model.History`/`model.Profile` (`model.go:11-54`) carry no LH, BBT, mucus, contraception, breastfeeding, skipped or pregnancy-test data. |
| 2.7 | Supported range + outliers | ✅ | `metrics.go:36-37,190-206` |
| 2.8 | Valid record (not skipped; period 2..10) | ⚠️ | Skipped cycles do not exist. A gap across a skip becomes a long outlier, which raises `long_cycle_outlier_detected`. The rule "9–10 may lower confidence" is not implemented. |
| 2.9 | Effective values, median-of-2 rounded half up | ✅ | `metrics.go:242-256` uses `phpround.Round`, half away from zero, so median(28,29)=29. A test exists at `metrics_test.go:143`. Exception: the onboarding seed writes a fake *real* period that feeds the median (see §2.40). |
| 2.10 | Sources (start, end, ovulation) | ⚠️ | start/end sources exist. `ovulation_source` exists in neither the enum nor the output. |
| 2.11 | Period-end model | ✅ | `resolver.go:115-121` |
| 2.12 | Soft/warning/hard caps | ✅ | `resolver.go:140-141`, from clinicalconfig `engine.warning_cap`/`period_hard_cap`. The daily card uses its *own* caps (`view/services.go:59-118`). |
| 2.13–2.15 | main_phase/subphase enums + mapping | ✅ | Generated from PHP (`zz_generated_main_phase.go`, `zz_generated_cycle_subphase.go`) |
| 2.16 | Next period; LH/BBT-anchored next; future_cycles k=1..6 | ❌ | Calendar only (`resolver.go:123`). No luteal-length median, no `future_cycles`. Future targets use an implicit roll-forward instead (`resolver.go:107-113`). |
| 2.17 | Ovulation priority BBT > LH > calendar, lh_bbt_conflict | ❌ | Always `next − 14` (`resolver.go:52,124`). BBT/LH priority lives only in `fertility/insights.go:309-322`, for finished-cycle history, with no conflict check. |
| 2.18 | Biological window O−5..O | ✅ in resolver / ⚠️ elsewhere | The resolver ends the window at O (`resolver.go:131`) and O+1 is `post_ovulation`/low (`resolver.go:219`). `view.Predict` (`services.go:52`), `legacy.InFertileWindow` (`legacy/engine.go:251-253`) and the clinicalconfig `after:1` field (`registry.go:89-90`) still use O+1. |
| 2.19 | Display window | ✅ | `resolver.go:126-132` |
| 2.20 | Resolve priority | ✅ | `resolver.go:155-248` |
| 2.21 | Menstrual tiers | ✅ (1 ambiguity) | `resolver.go:155-181`. Tier 21.5 emits only `…hard_cap_exceeded`, not `period_end_missing` (spec "+" is ambiguous). |
| 2.22 | period_expected tiers + suggest_pregnancy_test | ⚠️ | Tiers ✅ (`resolver.go:183-197`, `353-356`). `suggest_pregnancy_test` ❌. |
| 2.23 | Follicular 40/40/rest | ✅ | `resolver.go:437-462` |
| 2.24 | Fertile subphases | ✅ | `resolver.go:199-211` |
| 2.25 | Luteal priority + assignable split | ⚠️ (latent) | The code checks post_ovulation *before* pms/late (`resolver.go:218-227`), while the spec puts pms > late > post. This cannot show today (O+1 is always ≥ 9 days before next), but it will matter once LH/BBT can move O. The ceil/floor split ✅ (`resolver.go:465-479`). |
| 2.26 | fertility_level table | ⚠️ | `FertilityLevelV11` ✅ (`enums/cycle_logic.go:10-23`). Contraception suppression ❌. The daily card has its own level, read from the legacy subphase (`daily_card.go:277`). |
| 2.27 | Variability, regularity (FIGO, 12 mo), clinical_flags, teen | ⚠️/❌ | Variability ✅ (`metrics.go:258-264`). Regularity uses v1.1 semantics (`enums/cycle_logic.go:180-189`: last 3, fixed 7 days, different enum values). A second FIGO regularity lives in `analysis/cycle.go:94-140`. `clinical_flags` ❌. |
| 2.28 | data_quality | ⚠️ | Close (`resolver.go:300-318`). "good" also requires the period-duration source to be history. `insufficient` only comes when no anchor exists. |
| 2.29 | confidence + 14 reasons | ⚠️ | Missing from low: predicted_reference start, hormonal contraception, postpartum. Missing cap: age ≥ 45. Missing high alternative: `bbt_confirmed`. Reasons: 12 of 14 (`resolver.go:72-85`). |
| 2.30 | resolution_source | ✅ (mostly) | `resolver.go:250-258`. The spec's tie-breaker sentence conflicts with the onboarding override (see §3). |
| 2.31 | is_predicted | ✅ | `enums/cycle_logic.go:206-208` |
| 2.32 | warnings (20) | ❌ partly | 13 of 20 in the generated enum (`zz_generated_cycle_warning.go:12-24`). Missing: `stale_onboarding_anchor`, `hormonal_contraception_active`, `postpartum_cycles_not_returned`, `perimenopause_age_range`, `long_absence_no_period`, `skipped_cycle_recorded`, `lh_bbt_conflict`. |
| 2.33 | requires_user_input | ⚠️ | `enums/cycle_logic.go:193-199` has 4 of 6. Missing: `stale_onboarding_anchor`, `contradictory_anchor_data` (the latter is not in §2.32 at all). |
| 2.34 | days_to_ovulation nulls | ⚠️ | Null only when days_late > 14 (`resolver.go:353-356`). No suppression case exists. |
| 2.35 | Output shape | ⚠️ | Missing `anchors.ovulation_source`, `metrics.cycle_regularity`, `clinical_flags`, `future_cycles`, `suggest_pregnancy_test`. Extra legacy keys: 8 (`view/builder.go:102-112`). The worked example differs in `warnings`/`confidence_reasons`. |
| 2.36–2.38 | 21/28/35 tables | ✅ | 21 and 28 tested (`resolver_test.go:76-103,234-270`). 35 untested, though it passes by hand. |
| 2.39 | Confirm/recompute; skipped cycle | ⚠️/❌ | 1–7 ✅ by construction (pure recompute per request). 8 (skip) ❌: the card offers `period_not_started` (`daily_card.go:205`), but nothing stores it and the frontend never handles it. 9: cache invalidation, see the other report. |
| 2.40 | Forbidden rules | ⚠️ | Two violated: an assumed end is stored as a real end (onboarding seed), and O+1 is shown fertile in legacy paths. Contraception is not suppressed (input missing). |
| 2.41 | Stale onboarding anchor | ❌ | No rule. A 90-day-old LMP resolves `unknown` with days_late 62. |
| 2.42 | Limiting conditions | ❌ | None in the resolver. Pregnancy is handled outside it (mode switch). Teen/menopause only blank the card copy (`model.go:51-53`). |
| 2.43 | Today-screen mapping | ⚠️ | Split across the server daily card (`view/daily_card.go`, legacy rules) and the client `hero-state.ts`, each with thresholds that differ from §2.43. |

---

## 2. Findings per rule

### §2.3 Date, calendar, timezone
- Current: the engine works on `civildate.Date{Year,Month,Day}` (Gregorian). Jalali appears only in display layers. `Today(c Nower)` = `InTehran(c.Now())` (`platform/civildate/civildate.go:55-59`). The service takes `today` from the request clock (`cycle/service/service.go:61-62`).
- §6.1 "log at 00:30 Tehran, UTC server": ✅ already. `InTehran` converts the instant, so 2026-07-17T21:00Z → 2026-07-18 Tehran. Nothing pins this in a test.
- ⚠️ The device timezone is not used. A user in Europe/Berlin at 23:30 local on 07-17 (01:00 Tehran, 07-18) gets 07-18's status.
- Change: `today` becomes an explicit engine argument computed by the service from `user.timezone ?? "Asia/Tehran"`. Storage belongs to the other report. The engine itself never reads a clock.
- Tests: (a) instant 2026-07-17T21:00:00Z, tz Asia/Tehran → today 2026-07-18; (b) same instant, tz Europe/Berlin → 2026-07-17; (c) instant 2026-03-20T20:29Z (Nowruz boundary, Esfand of a leap year 1403 → 1404-01-01) → Tehran day 2026-03-21, and cycle_day continuity across it.

### §2.4–2.5
✅ `DiffDays` and `cycleDay = diff + 1` (`resolver.go:134`). There is no zero-based path. `test_cycle_day_is_one_based` exists. Performance: `AddDays`/`DiffDays` call `time.Date`/`AddDate` on every use (`civildate.go:101-107`). See §4.

### §2.6 Inputs
❌ for fertility markers and context. The resolver signature is `Resolve(histories, profile, target, today, metrics)` (`resolver.go:97`). Birthday sits on `model.Profile` (`model.go:44`) but the resolver never reads it. See the other report for the loaders.

### §2.7–2.8 Valid records
- ✅ Cycle 21..45 and period 2..10 (`metrics.go:36-39,190-221`). Outliers only set flags.
- ⚠️ Skipped cycles: a skipped month spans two real starts about 2L apart, e.g. starts 01-01 and 02-26 = 56 days. The pair becomes a long outlier, so it is (correctly) excluded from the median, but `long_cycle_outlier_detected` is wrongly emitted (`resolver.go:272-274`) and `OnlyOutlierHistory` can force `poor`. v6: a cycle containing a recorded skip is neither valid nor an outlier, and it raises `skipped_cycle_recorded` instead.
- ⚠️ Period 9–10 "may lower confidence": no rule (spec gap, see §3).
- Tests: starts [01-01, 01-29, 02-26] + skip recorded at 03-26 + start 04-23 → valid lengths [28,28], the 56-day gap is excluded, no long-outlier warning, `skipped_cycle_recorded` is present.

### §2.9 Effective values
- ✅ `median` (`metrics.go:242-256`): an even count averages the middle pair and rounds half away from zero, so median(28,29)=29 and median(27,28)=28 (27.5 → 28). Last ≤3 valid records, fallback declared → 28/5 (`metrics.go:223-231`). Covered by `metrics_test.go:45,67,143`.
- A duplicate median exists in `cycle/insights/history.go:194-206` (same rounding).
- ⚠️ `LengthsManual` (B-N1-09, `metrics.go:161-164,234-239`) puts the profile first. This is not in v6, so it is a product decision (§3).
- ⚠️ Contamination, a §2.40 violation: the onboarding seed row (`profile/service.go:480-497`) stores `period_end = LMP + declared − 1` with `is_confirmed=true`. That declared duration then counts as a *real* valid period length in `validPeriodDurations`. Example: onboarding LMP 07-01 with declared period 5, then one real logged period 07-29..08-04 (7 days). Today: median(5,7)=6. v6 (declared excluded): 7.
- Tests: [28,29] → 29; [27,28] → 28; [28,30,35] → 30; [] + declared 31 → 31 (source declared); [] + nothing → 28/5 (default); onboarding seed + one real 7-day period → effective_period_length 7.

### §2.10 Sources
✅ `current_period_start_source` (4 values) and `current_period_end_source`. `current_period_end_is_confirmed` is always present (`status.go:39-48`). ❌ `ovulation_source` (bbt_confirmed | lh_positive | calendar | unknown) appears in no enum and no output. It must be `unknown` when there is no anchor or when contraception suppresses it.

### §2.11–2.12 End model + caps
✅ `resolver.go:115-121,140-143`. Caps are read from clinicalconfig (defaults 12, max(EPL+3, 8), 14 late). ⚠️ The view's `EvaluateOpenPeriod` (`view/services.go:90-118`) hardcodes 12/8/3 and uses the **profile** period duration instead of the effective one. Example: profile period 7, history median 4. The resolver gives warning_cap = max(4+3, 8) = 8, so day 9 is `menstrual_possible` + `period_end_missing_warning_cap_exceeded` + poor. The card gives maxExpected = max(7+3, 8) = 10, so day 9 reads «روز ۹ پریود» with no "end overdue" prompt. v6 should have one source: the card reads the status.

### §2.16 Next period / future cycles ❌
- Current: `predictedNext = start + ecl` (`resolver.go:123`). No LH/BBT path, no `effective_luteal_length`.
- Future targets (`target > today`) roll the anchor forward in whole cycles (`resolver.go:107-113`) and set `predicted_reference`. This roll-forward is the only "future cycle" mechanism. It is not v6's `future_cycles[k]`, and it misbehaves when the user is late. Example: anchor 01-01, L=28, P=5, today 02-01 (days_late 3, `period_expected`). Target 02-02 rolls to start 01-29 (a *past* date), cycle_day 5, so main_phase is `menstrual` (`resolver.go:175-176`). The calendar would show "period day 5" tomorrow while today shows "3 days late". v6 needs an explicit rule for late users (§3, item 4).
- Confidence: under v1.1 the roll-forward gives medium. Under v6 §2.29, a predicted_reference start gives **low**, which conflicts with "k≥2 at most medium" (§3, item 3).
- Change: compute `future_cycles[1..6]` as a pure display chain from `predicted_next_period_start` (each item: period_start, period_end, fertile_start/end, ovulation, confidence). Future target dates resolve against that chain. Nothing is ever stored, and the legacy `cycle_calculations` table must not receive it.
- BBT/LH: `next = ovulation + effective_luteal_length`, where effective_luteal_length = median of (next real start − BBT ovulation) over completed cycles, keeping values in 10..16, else 14. Round with the same half-up rule as §2.9 (spec silent, §3).
- Tests: L=29, start 07-01 → k=1 07-30, k=2 08-28 (ov 08-14), k=6 12-24 (07-30 + 5×29 = 145 days). Confidence for k=1 ≤ the status confidence, for k≥2 ≤ medium. BBT luteal history [12,13] → luteal 13 (12.5 rounded up), BBT ov 07-14 → next 07-27. Luteal history [9,17] → both invalid → 14. LH positive 07-13 (ov 07-14, no BBT) → next 07-28.

### §2.17 Ovulation ❌
- The resolver: always calendar (`resolver.go:124`).
- `internal/fertility/bbt/bbt.go` vs v6:
  - Rise rule differs. bbt.go needs the **first** high reading ≥ coverline + 0.20 °C and the next two > coverline (`bbt.go:24,159-178`). v6 standard: 3 consecutive readings > max of the previous 6. v6 strict: the **third** reading ≥ +0.2 °C. Example: baseline max 36.50, then 36.55, 36.60, 36.72. v6 standard confirms (ov = day before 36.55). v6 strict also confirms (36.72 ≥ 36.70). bbt.go does not confirm, because 36.55 < 36.70 and it waits for a later start.
  - Minimum readings: bbt.go needs 6 + 3 = 9 readings. v6 requires ≥ 10 readings in the cycle.
  - ✅ Ovulation = day before the first high reading (`bbt.go:96-102`). Temperatures are integer hundredths (good: no float noise).
  - "Consecutive" in bbt.go means consecutive *readings*, skipping days without one (`bbt.go:145-147`). v6 does not say whether a missing day breaks the run (§3).
  - Coupling: `bbt.CycleInput.FertileWindow` is filled *from the cycle engine* (`fertility/chart.go:85-88,94-110`, up to 60 resolver calls per cycle). If the engine consumes BBT, the dependency becomes circular. The shift detector must move into (or under) the engine package, and fertility then reads it from there.
- LH: `fertility/insights.go:228-251` takes the first `positive` (and also `faint`, which is not in v6) and uses the +1 day (`insights.go:314-316`, `DiffDays+2` = cycle day of first positive + 1). That logic is history-only, not in the engine.
- `lh_bbt_conflict` ❌ anywhere.
- Tests (cycle start 07-01): BBT 6 baseline days ≤ 36.50 on days 8–13, then 36.62/36.65/36.72 on days 14–16 with ≥ 10 readings → bbt_confirmed, O = day 13. Same with only 9 readings → not confirmed → falls to LH or calendar. LH positive day 12 → O = day 13, source lh_positive. LH day 10 (O=11) + BBT O=14 → |3| > 2 → O=14, warning `lh_bbt_conflict`. LH O=12 + BBT O=14 → |2| → no conflict, BBT wins. Strict-mode case: 36.51/36.52/36.60 over a 36.50 max → standard confirms, strict does not.

### §2.18–2.19 Fertile windows
- ✅ Resolver: `biologicalFertileStart = O − before(5)` and `displayFertileEnd = ovulation` (`resolver.go:125-132`). O+1 goes to luteal `post_ovulation` (`resolver.go:214-220`) with fertility low. ✅ "menstrual overrides fertile", by priority.
- ⚠️ Still O+1 elsewhere: `view.Predict` → `predictions.fertile_window_end` (`view/services.go:52`), `legacy.InFertileWindow` (`legacy/engine.go:251-253`, which drives `/cycle/month` `is_fertile_window`), the legacy `PhaseFor` "ovulation" phase through O+1 (`resolver/phase_mapper.go:17-18`), and clinicalconfig `engine.fertile_window.after` default 1 (`clinicalconfig/registry.go:89-90`). v6: drop the `after` field, or pin it to 0 with a new version. The registry source string also still says "v1.1 (task.md §2.12)" (`registry.go:75`).
- Tests: 28-day cycle with real end day 5. Day 15 = ovulation_likely/peak. Day 16 = luteal/post_ovulation/low, `predictions.fertile_window_end` = day 15, and the calendar day 16 has `is_fertile_window=false`. 21-day cycle with a 7-day period → display window day 8 only (O=8) → day 8 ovulation_likely. 21-day with an 8-day confirmed period → window empty, day 8 menstrual, day 9 post_ovulation.

### §2.20–2.21 Priority and menstrual
✅ All five tiers (`resolver.go:155-181`). Hand-checked with EPL=5: soft 5, warning 8, hard 12. Day 6 → menstrual_possible + `period_end_missing`. Day 9 → + `…warning_cap_exceeded`, poor, low, input. Day 13 → resolves onward (follicular/fertile), + `…hard_cap_exceeded`, poor, low (via `pastWarningCapOpen`, `resolver.go:143,329`), reason `missing_period_end_beyond_hard_cap`. Existing tests: `resolver_test.go:127-170`.
⚠️ Product note (spec-faithful, but heavy): with an open period, *every later day of that cycle* stays poor/low/requires_input, because the `isRealOpenPeriod` anchor has no end. In practice healthlog closes periods from bleeding logs (`db/queries/healthlog/side_effects.sql:37`), which softens this.
§6.1 test "open period up to day 13 and after hard cap": add day 13 *and* day 20 (fertile/luteal, still poor, reasons include beyond_hard_cap).

### §2.22 period_expected
- ✅ Tiers (`resolver.go:183-197`): 0–7 calm, 8–14 overdue + low, >14 unknown + both warnings + poor + low + input. Anchors are kept, `days_to_period` = max(0, raw) = 0, `days_to_ovulation` nil (`resolver.go:353-356`), fertility unknown (`resolver.go:349-352`). Tests exist for 7/8/20 (`resolver_test.go:171-207`). §6.1 asks for **15**: add it (15 → unknown, 14 → period_expected).
- `resolution_source` for >14 is `prediction` in code. The spec does not say (§3).
- ❌ `suggest_pregnancy_test`. Rule: mode ∈ {cycle, ttc} ∧ !hormonal ∧ (ttc ? days_late ≥ 1 : days_late ≥ 8). Tests: ttc+1 → true; cycle+7 → false; cycle+8 → true; cycle+8 with pill → false; pregnancy mode → false; days_late 0 → false.

### §2.23–2.25 Follicular, fertile, luteal
- ✅ Follicular split: `early = max(1, ⌊0.4L⌋)`, `mid = max(1, ⌊0.4L⌋)`, late = rest (`resolver.go:452-461`). L=1 → mid, L=2 → early/late, L=3 → 1/1/1, L=4 → 1/1/2, L=11 → 4/4/3 (35-day table days 6–9/10–13/14–16).
- ✅ Fertile: dto ≥ 3 → rising, 1–2 → high, 0 → ovulation_likely (`resolver.go:202-209`). If `before` were configured above 5, days 6–7 would also read "rising". Acceptable, but the v6 spec pins 5,4,3.
- ⚠️ Luteal priority order (`resolver.go:218-227`): the code is post_ovulation → pms → late → early/mid, the spec is pms → late → post → early/mid. Unreachable with a calendar O (days_until(O+1) = 13) or with `next = O + luteal` where luteal ≥ 10 (days_until = 9). Fix it anyway for spec fidelity. Test: inject O such that O+1 = next−3 → pms_possible.
- ✅ Assignable O+2..next−7, early = ⌈len/2⌉ (`resolver.go:465-479`). Days outside the range (when the range is empty) fall to mid_luteal. Tests: luteal 14 (normal) → 6-day range 3/3. "Very short assignable range": with BBT/LH O and luteal 10 → range O+2..O+3 = 2 days → 1 early, 1 mid. Luteal 9 is impossible (invalid, falls back to 14). With an empty range (end < start, e.g. luteal 8 when injected), days from O+2 to next−7 do not exist and every non-post/late/pms day becomes mid_luteal.

### §2.26 fertility_level
✅ The table matches `FertilityLevelV11` (`enums/cycle_logic.go:10-23`), and unknown main_phase forces unknown (`resolver.go:349-352`).
❌ Suppression (hormonal contraception, §2.42) → `unknown` for every subphase.
⚠️ The daily card shows `sub.FertilityLevelV11()` of the **legacy** subphase (`view/daily_card.go:277`, fed from `base.Subphase()`, `builder.go:141`) and hardcodes `low` for the overdue and due cards (`daily_card.go:180-225`), whereas the spec gives `unknown` for period_expected. Example: day 29 of a 28-day cycle: `cycle_view.fertility_level=unknown` but `daily_card.fertility_level=low`.

### §2.27 Variability, regularity, clinical flags
- ✅ `cycle_variability` = max−min of the last 3 valid, null with fewer than 2 (`metrics.go:258-264`). Used only for confidence (>5).
- ⚠️ `cycle_regularity`: the engine has `RegularityStatus` (`not_enough_data|relatively_regular|irregular_possible`) over the last 3 valid lengths with a fixed 7 days (`enums/cycle_logic.go:180-189`). The analysis tab has a second, FIGO age-aware implementation over a date range with single-outlier exclusion (`analysis/cycle.go:94-140`, `analysis/analysis.go:88-105`). v6 wants one value: `regular|irregular|insufficient_data` over **real complete** cycles whose start is in the last 12 months, at least 3, skips excluded, threshold by age (26–41 → ≤ 7; 18–25 and 42–45 → ≤ 9).
  - Concrete difference: age 22, cycles [26,30,34] (range 8). v1.1 gives `irregular_possible`, v6 gives `regular` (≤ 9).
  - Another: 5 cycles in 12 months [28,28,28,37,28]. v1.1 last 3 [28,37,28] → range 9 → irregular_possible. v6 range 9, adult → irregular. Same verdict, but over a different window.
  - The analysis `markOutlier` (one-off exclusion) is not in v6 (§3).
- ❌ `clinical_flags`:
  - `frequency`: from effective_cycle_length, <24 frequent / 24–38 normal / >38 infrequent.
  - `duration`: real period > 8 → prolonged.
  - `absent`: 90 days without a period and mode ∉ {pregnancy, postpartum}.
  - Teen (<18 or ≤ 3 years since menarche): the normal range is 21–45, so frequent < 21 and infrequent > 45.
  - The thresholds already exist as clinicalconfig `cycle.figo.*` (`registry.go:91-95`) but are only used by the analysis tab.
- Tests: age 30 [27,29,33] → regular (6). Age 30 [26,30,34] → irregular (8). Age 22 same → regular. 2 cycles → insufficient_data. 3 cycles of which one is skipped → insufficient_data. Cycles older than 12 months are ignored. eff 23 → frequent. eff 39 → infrequent. Teen eff 23 → normal. Last real period 9 days → prolonged. Last start 91 days ago, mode cycle → absent=true + `long_absence_no_period`. Same in postpartum → false.

### §2.28 data_quality
- ✅ Order insufficient → poor → good → partial (`resolver.go:304-318`). poor = forced / past warning cap open / late > 14 / contradiction / only outliers.
- ⚠️ "good" adds `CycleLengthSource == recent && PeriodDurationSource == recent` (`resolver.go:312-313`). The spec says "≥ 2 real valid records + anchors complete". Example: 3 valid cycles (2 lengths) but no closed period in history (all ends via healthlog missing). Code → partial (period source falls to profile/default). Spec → arguably good if "anchors complete" is read loosely. Keep the code's stricter reading, but make it explicit (§3).
- ⚠️ `insufficient` only comes from the no-anchor path (`resolver.go:416-435`). The spec's "onboarding/fallback gives no sufficient anchor" maps to the same thing.
- Tests: the §2.35 example → good. Onboarding-only → partial. 1 real cycle → partial. Open past day 8 → poor. days_late 15 → poor. Only outlier gaps [18,50] → poor. No anchor → insufficient.

### §2.29 Confidence
Current low conditions (`resolver.go:325-333`): forced, main unknown, late > 7, poor, past warning cap, valid < 2, var > 5, default/onboarding resolution.
Missing:
- low when `current_period_start_source = predicted_reference`;
- low when `hormonal_contraception`;
- low when postpartum with fewer than 2 real periods;
- cap at **medium** for age ≥ 45 (Birthday is in `model.Profile` but never read) and for future_cycles k ≥ 2;
- the high alternative "variability ≤ 5 **or** ovulation_source = bbt_confirmed" (`resolver.go:339` hard-requires rng ≤ 5).

`confidence_reasons`: 12 constants (`resolver.go:72-85`). Missing `stale_onboarding_anchor` and `hormonal_contraception_active`.

Tests:
- 3 cycles [28,28,35] (var 7) with BBT confirmed this cycle → v6 gives high only if not low. But var > 5 is a *low* condition, which beats high, so low wins. The bbt alternative in "high" is dead under the stated order (§3).
- Age 46 with a perfect history → medium + `perimenopause_age_range`.
- Pill → low + reason `hormonal_contraception_active`.
- Future k=2 → ≤ medium.

### §2.30–2.31 Resolution source, is_predicted
✅ user_logged inside a confirmed period. `user_logged_with_assumed_end` for an open or onboarding-in-soft-cap period with a real start (`resolver.go:162-179`). prediction for the rest. Onboarding override → `onboarding_based`, or `default_based` when both lengths are default (`resolver.go:250-258,482-487`). The §2.35 example gives `prediction` ✅. is_predicted ✅.
Note: the onboarding override skips `period_expected` (onboarding days 0–14 late stay `prediction`). That is v1.1 behaviour, and v6 does not decide it (§3).

### §2.32–2.33 Warnings and requires_user_input
- Enum: 13 of 20, generated from **PHP** (`zz_generated_cycle_warning.go:1`, `cmd/enumgen`). Adding v6 values means touching the frozen Laravel enums or leaving the generator. Recommendation: hand-written v6 enums in the new engine package.
- `prediction_based_output` is emitted whenever resolution = prediction (`resolver.go:286-288`), which includes the §2.35 example, where the spec shows `warnings: []` (§3, item 1). `resolver_test.go:124` asserts it is present.
- `RequiresUserInput()` (`enums/cycle_logic.go:193-199`): 4 values. v6 adds `stale_onboarding_anchor` and `contradictory_anchor_data`.

### §2.34 days_to_ovulation
⚠️ Null only when days_late > max (`resolver.go:353-356`). Add null for: ovulation not computable (no anchor, which already holds through `unresolved`), suppression by contraception, and postpartum-before-return. Tests: pill → null. Late 15 → null. Late 14 → −(14+14) = −28.

### §2.35 Output shape: §2.35 vs today's `cycle_view`
Today's `cycle_view` = `StatusAPI` (`resolver/status.go:59-77`) + 8 legacy keys (`view/builder.go:102-112`): `phase`, `data_status`, `predictions`, `profile_values`, `calculated_values`, `effective_values`, `data_quality_details`, `daily_card`.

| §2.35 field | Today |
|---|---|
| date, cycle_day, main_phase, subphase, fertility_level, days_to_ovulation, days_to_period, days_late | ✅ same names and types |
| anchors.{current_period_start/_source, current_period_end/_source/_is_confirmed, predicted_next_period_start, estimated_ovulation_date} | ✅ |
| anchors.ovulation_source | ❌ |
| metrics.{effective_cycle_length, effective_period_length, cycle_variability} | ✅ |
| metrics.cycle_regularity | ❌ (`data_quality_details.regularity_status` holds v1.1 values) |
| clinical_flags {frequency, duration, absent} | ❌ |
| future_cycles [...] | ❌ |
| suggest_pregnancy_test | ❌ |
| confidence, confidence_reasons, data_quality, resolution_source, is_predicted, requires_user_input, warnings | ✅ (contents differ as above) |

There are two `confidence` notions in the payload: §2.29 `confidence`, and `predictions.confidence`/`data_quality_details.confidence` from the v1.1 `CycleConfidenceCalculator` (`view/services.go:141-165`). They can disagree. Example: 3 regular cycles while the user is 8 days late gives top-level `low` and `predictions.confidence` `high`. The frontend reads both (`frontend/src/entities/cycle/api/schema.ts:293-330`).

**Worked example by hand** (periods 07-01..05 confirmed, 3+ prior cycles giving eff 29, var 3, ≥ 2 closed periods of 5; target = today = 07-18):
start 07-01 (user_logged), end 07-05 confirmed. next = 07-30. O = 07-16. Bio window 07-11..16, display 07-11..16. cycle_day 18. dtp 12. dto −2. Not menstrual, not expected, not fertile. Luteal: days_since 2, days_until 12, assignable 07-18..07-23 (6 days), early = 3 → `early_luteal`, low. Resolution `prediction`, data_quality good, confidence high (rng 3, ≥ 2 durations).
**Mismatches with the spec example:** the code emits `warnings: ["prediction_based_output"]` and `confidence_reasons: ["prediction_based_output"]` where the spec shows `[]`, and it lacks `ovulation_source`, `cycle_regularity`, `clinical_flags`, `future_cycles` and `suggest_pregnancy_test`. Everything else matches (spec dates are correct: Tir has 31 days, so 1405-04-27 = 2026-07-18).

### §2.36–2.38 Tables
21 and 28 pass existing tests. 35 hand-checked: O = 22, window 17–22, follicular 6–16 split 4/4/3, early luteal 24–26, mid 27–29, late 30–32, pms 33–35, expected 36. Add a table-driven test that encodes all three columns verbatim.

### §2.39 Confirm / recompute
1–7 ✅ by construction: the engine is recomputed from the stored history per request, and predictions are never persisted (no INSERT of predicted rows: `db/queries/cycle/periods.sql:49`, `profile/cycle_history.sql:8-10` are real or onboarding writes).
8 ❌ Skipped cycle: no table, no endpoint, and the frontend has no handler for `period_not_started` (grep of `frontend/src` finds no match).
- Required engine behaviour: `next' = previous predicted next + L`, the skip excluded from medians and regularity, warning `skipped_cycle_recorded`.
- Test: start 07-01, L=28, skip on 07-29 → next 08-26. On 08-10: v1.1 says days_late 12; v6 says cycle continues (see §3 for cycle_day).
9: recompute on past edits. The engine is pure, so this is a cache concern (other report: cache key).

### §2.40 Forbidden rules, one by one
1. Predicted start in period_logs: ✅ none.
2. Assumed end stored as a real end: ⚠️ **violated** by the onboarding seed. `profile/service.go:481-497` writes `period_end_date = LMP + declared − 1`, `bleeding_length`, `is_confirmed=true`, `source=user_profile_confirmed`.
3. ovulation / menstrual_possible as main_phase: ✅ never (`resolver.go`). The legacy `phase=ovulation` key survives in `cycle_view.phase` (`enums/cycle_logic.go:145-146`, MainPhase fertile → legacy "ovulation"). That is legacy, but it is a top-level field of the same payload.
4. Fertile on menstrual days: ✅ resolver. Legacy uses `isFertile && !isPeriodDay` (`legacy/engine.go:107`), ✅.
5. 0-based cycle_day: ✅.
6. Start/end without source, end without is_confirmed: ✅ in §35 anchors. ⚠️ `predictions.next_period_end` (`view/builder.go:81-89`) is an end with no source or confirmation flag. It is a prediction, so arguably exempt; it disappears with the legacy block anyway.
7. Predicted data in effective values: ✅ predictions never enter. ⚠️ Declared values do enter via the confirmed onboarding seed (§2.9).
8. Jalali strings, 30-day months, UTC server: ✅ engine. TZ is fixed to Tehran, not the device.
9. Fertile window or fertility_level under hormonal contraception: ❌ not suppressed (input missing).
10. 21–45 as "normal" in the UI: frontend audit (not in scope). The engine exposes no "normal" label today.
11. Storing future_cycles: ✅ n/a today. Keep legacy `cycle_calculations` (dead storage) away from it.

### §2.41 Stale onboarding anchor ❌
Example: onboarding anchor 07-01 (estimated row or profile LMP), no logs, profile 28/5, target = today = 09-29 (diff 90 > 28 + 14).
- v1.1: next 07-29, days_late 62 → `unknown`, `predicted_period_overdue` + `cycle_unresolved_after_expected_period`, poor, low, dto null.
- v6: n = ⌊90/28⌋ = 3 → start 09-23 (`predicted_reference`), cycle_day 7. Assumed end 09-27. Follicular 09-28..10-01 (4 days: 1/1/2) → 09-29 = `mid_follicular`. Low, `stale_onboarding_anchor`, requires_input.

Caveat: the trigger requires "no actual period_log". Because onboarding is stored as a confirmed `user_profile_confirmed` row (other report), the rule never fires until onboarding rows are tagged `onboarding_declared`.
Tests: diff 42 (= L+14) → no stale (period_expected, late 14). Diff 43 → stale, n=1. Diff 90 as above. Any real log → no stale.

### §2.42 Limiting conditions ❌ (all)
- Hormonal contraception: method ∈ {combined_pill, progestin_pill, hormonal_iud, injection, implant, ring, patch} (from `contraception_methods.method`, `db/migrations/00017_contraception.sql`; copper_iud is not hormonal). Effect: no fertile/ovulation, fertility unknown, dto null, ovulation_source unknown, warning + reason, low.
- Pregnancy: the engine returns a "suspended" status (or the service never calls it). Today mode switching is handled outside.
- Postpartum/breastfeeding: no predictions until 2 real postpartum periods, plus warning.
- Positive test: prompt flag.
- Age ≥ 45: medium cap + warning.
- < 18: teen flags.

Tests are listed in §6.1: pill; postpartum without a period; age 46; age 16.

### §2.43 Today-screen mapping
Partly server-side, but not as specified:
- Server: `daily_card` (`view/daily_card.go`) built from the **legacy** cycle_day/subphase (`builder.go:121-149`). Its own prediction is `view.Predict`, anchored at the last confirmed start ≤ *today* (not the selected date) with O−5..O+1 (`builder.go:126-127`). Its own open-period caps come from `EvaluateOpenPeriod`. Actions: `confirm_period_start`, `period_not_started`, `log_period_end`, `still_bleeding`.
- Client: `frontend/src/screens/home/model/hero-state.ts:40-59`:
  - start prompt at `days_to_period ∈ 0..2` or any `days_late > 0`. Spec: predicted_next − 3 .. + 14, and only while no period is open.
  - end prompt on every open-period day. Spec: days_since ≥ EPL − 1 .. hard_cap.
  - "end missing" banner at cycle_day > periodLength. Spec: only on warnings `…warning_cap_exceeded`/`…hard_cap_exceeded`.
  - No overdue banner gated at ≥ 8 days, no stale-anchor banner, no single-banner priority.
- Recommendation: the engine emits a small pure `today_ui` block from the status (ring_action: start|end|edit, banner: end_missing|overdue|ask_last_period|null, prediction_card ranges: period, PMS, fertile display window, ovulation+source, cycle length + regularity). Frontend and daily card then read it instead of re-deriving it (§3, item 17).
- Tests: next−3 → start prompt. next+14 → start prompt. next+15 → edit. Open period day EPL−1 → end prompt. Day hard_cap+1 → edit + end_missing banner. days_late 8 with ttc → overdue banner + test button. Stale anchor → ask_last_period. Several conditions → only the first banner in table order.

---

## 3. Spec inconsistencies / gaps that need a user decision (with recommended default)

1. **§2.35 example vs §2.32 `prediction_based_output`.** The example resolves via `prediction` but shows `warnings: []` and `confidence_reasons: []`. The code (and `resolver_test.go:124`) emits `prediction_based_output` there. *Recommend:* emit `prediction_based_output` only when the **anchor** itself is predicted (`predicted_reference`) or the output is a future-cycle projection, not for ordinary prediction-resolved phases. The example then holds exactly.
2. **§2.33 lists `contradictory_anchor_data`, absent from the §2.32 enum**, and no rule produces it. *Recommend:* add it to §2.32 and raise it for impossible anchors: end < start, overlapping real periods, a real start inside another real period. These imply poor and requires_input.
3. **Future target dates:** §2.29 makes a `predicted_reference` start **low**, while §2.16 says future cycles k ≥ 2 are "at most medium" (implying medium is reachable). *Recommend:* future dates resolve against `future_cycles[k]`. Confidence = min(base confidence, k=1 ? base : medium). Reserve `predicted_reference` + low for the §2.41 stale-anchor case.
4. **Future chain while late:** §2.16 chains from `predicted_next_period_start`, which is already in the past when days_late > 0. *Recommend:* while late, k=1 starts at `max(predicted_next, today + 1)`, and the calendar shows the remaining days as period_expected up to that point. Alternative: keep the stale chain and mark it overdue. Avoid v1.1's "period day 5 tomorrow" artefact.
5. **`future_cycles` item shape.** The example shows only `{k:2, period_start, ovulation}` and omits k=1. *Recommend:* k=1..6 always, each `{k, period_start, period_end, fertile_start, fertile_end, ovulation, ovulation_source:"calendar"|…, confidence}`.
6. **§2.27 frequency from `effective_cycle_length`**, but §2.8 excludes cycles < 21 and > 45 from it. Real 18-day cycles fall back to declared or 28 and read "normal". *Recommend:* frequency from the median of the last 3 *real complete* cycle lengths (outliers included, skips excluded). Fall back to effective_cycle_length when there are fewer than 1.
7. **§2.27 duration** ("real period_length > 8": which period?) **and absent** (90 days from what, `>` or `≥`). *Recommend:* duration = prolonged if the latest closed real period, or the median of the last 3 closed periods, is > 8 (pick one; I recommend the median of the last 3). absent = `difference_in_days(today, last real period_start) ≥ 90` and the user is not currently in a period.
8. **Teen definition** needs menarche age or date, which the app does not collect. Regularity thresholds for < 18 and > 45 are undefined. *Recommend:* teen = age < 18 only, until menarche is collected. For < 18 and > 45, regularity uses ≤ 9 days, or reports `insufficient_data` with the teen range applied to frequency.
9. **Regularity "without skipped" and outliers:** does a skip disqualify the window or just get excluded? Are cycles < 21 or > 45 included? *Recommend:* exclude skipped cycles, include outliers (they are real variation), and drop the analysis-tab one-outlier exclusion (`analysis/cycle.go` `markOutlier`) so the tab and the engine agree.
10. **§2.30 "enum order is only a tie-breaker"** would put `prediction` above `onboarding_based`, making `onboarding_based` almost unreachable. The code applies an onboarding override instead (skipping period_expected). *Recommend:* keep the override (onboarding anchor → `onboarding_based`/`default_based` for all phases incl. period_expected 0–14). Read the tie-breaker as applying only to genuinely equal sources.
11. **§2.22 > 14 late `resolution_source`** is unspecified (code: `prediction`). *Recommend:* `prediction`, so confidence is low, not unknown, matching the "not an invalid output" sentence.
12. **§2.21.5** "+ hard_cap_exceeded": cumulative with `period_end_missing` or not? *Recommend:* cumulative (`period_end_missing`, `…warning_cap_exceeded`, `…hard_cap_exceeded`), so banners keyed on any of them keep working.
13. **Hormonal contraception phase model:** if ovulation is not computed, luteal/fertile are undefined. *Recommend:* main_phase ∈ {menstrual, period_expected ("withdrawal bleed"), follicular for all other days}, subphase `mid_follicular`, fertility `unknown`, ovulation fields null, ovulation_source `unknown`. Alternative: `unknown` main_phase, which would wrongly trigger insufficient-anchor semantics.
14. **Postpartum / breastfeeding before 2 real periods:** output unspecified. *Recommend:* real logged periods resolve as menstrual. All other days give main_phase `unknown`, resolution `unknown` → confidence unknown, warning `postpartum_cycles_not_returned`, no future_cycles, requires_input false. Also decide whether breastfeeding without postpartum mode triggers it.
15. **Skipped cycle semantics:** after "I had no period", what are `current_period_start` and `cycle_day`? Does the count continue (day 40+) or restart at the skipped date? How long does `skipped_cycle_recorded` stay? When may a user skip (any day_late ≥ 0, or only after N days)? Can a skip be undone? *Recommend:* the skip creates a virtual anchor at the skipped predicted date with `current_period_start_source = predicted_reference` (low, which is honest), cycle_day restarts there, and the warning stays until the next real start. A skip is allowed when days_late ≥ 0 and is deletable.
16. **BBT rule variant:** standard vs strict, which is the default? Is a missing day a break in "consecutive"? Does `faint` LH count? *Recommend:* strict (third high ≥ +0.20 °C) as default via clinicalconfig `engine.bbt.strict=1`, readings consecutive by reading (gaps allowed, matching bbt.go), ≥ 10 readings, and `faint` ignored for `lh_positive`.
17. **§2.43 ownership:** is the today mapping engine output or UI logic? *Recommend:* the engine emits a pure `today_ui` block so the web, the daily card and future clients cannot drift. "هنوز نه"/"هنوز ادامه دارد" stay client-local per spec.
18. **§2.8 periods of 9–10 "may lower confidence depending on history":** no rule. *Recommend:* if the latest closed period is 9–10 and the user's median is ≤ 7 → add reason `low_history`? Simpler: none in v6.0, documented.
19. **§2.29 order makes the high alternative "or ovulation_source = bbt_confirmed" dead**, because variability > 5 is already a low condition that wins first. *Recommend:* make the low condition read "cycle_variability > 5 **and** ovulation_source ≠ bbt_confirmed".
20. **B-N1-09 manual lengths switch** (profile wins over history) is not in v6. *Recommend:* keep it as an explicit user override. Resolution and data_quality treat it as declared (partial / no high), as the code does today.
21. **§2.3 device timezone:** no field stores it. *Recommend:* `user_profiles.timezone` (IANA, default Asia/Tehran), sent by the client. See the other report.
22. **Product risk, spec-faithful:** an open period past warning_cap keeps the whole cycle poor/low/requires_input. Confirm this is intended, or limit it to days ≤ hard_cap + N.

---

## 4. Architecture and performance

**Coverage (run 2026-10-21):**
```
ok  internal/cycle/cache           coverage: 87.1%
ok  internal/cycle/insights        coverage: 73.3%
ok  internal/cycle/legacy          coverage: 98.4%
ok  internal/cycle/metrics         coverage: 97.6%
ok  internal/cycle/model           coverage: 75.0%
ok  internal/cycle/periods         coverage: 0.0%   (integration-only)
ok  internal/cycle/recommendation  coverage: 90.2%
ok  internal/cycle/resolver        coverage: 94.5%
ok  internal/cycle/service         coverage: 0.0%   (integration-only)
    internal/cycle/store           coverage: 0.0%
ok  internal/cycle/view            coverage: 97.4%
```
No `Benchmark*` exists in `internal/cycle` or `internal/fertility/bbt`.

**Purity:**
- `metrics` and `daily_card` are pure.
- `resolver.capsOn` (`resolver.go:61-69`) and `view.Predict` (`services.go:51-52`) read the **package-global** clinicalconfig snapshot, five lookups per Resolve. `clinicalconfig.current()` may spawn a background reload goroutine on a lookup (`clinicalconfig/store.go:123-126`). So the "pure" engine has hidden global state and a side effect, and tests depend on the global default.
- The clock is injected (`today` is a parameter) ✅.

**Cost:**
- Each `Resolve` scans all histories (`anchorFor`, `resolver.go:389-414`, O(H)) and does about 15 `civildate` AddDays/DiffDays. Each of those builds a `time.Time` (`civildate.go:101-107`), plus small slice allocations (warnings, reasons) and four `*int`.
- `view.Build` recomputes `metrics.Calculate` (sort + clones) per day (`builder.go:117`).
- Hot callers:
  - `fertility/chart.go:94-110` runs up to 60 resolves per BBT cycle × `maxBBTCycles`, plus 1.
  - `fertility/insights.go:283-302` runs up to 13 per request.
  - `/cycle/month` runs the legacy engine for 28–31 days (no resolver).
  - A v6 calendar with the resolver per day would cost about 31 × O(H) plus allocations. That is fine in absolute terms (microseconds), but wasteful and growing with history.

**Tests to re-record or retire:**
- Laravel-recorded goldens: `contract/golden/cycle-sweep` (102 files, 14 MB; 34 contain `main_phase`), `contract/golden/cycle` (79), `period` (6), `companion` (2), `companion-home` (1), `teen` (2) → **124 golden files** contain `main_phase`/`cycle_view`.
- `view/golden_sweep_test.go` and `legacy/golden_sweep_test.go` replay them with semantic equality, plus allowlists in `contract/allowlist/cycle-sweep.yaml`/`cycle.yaml`.
- Every v6 change to warnings, confidence, new fields or O+1 breaks them. They cannot be re-recorded from Laravel (prod Laravel stays v1.1, no cutover).
- Plan: freeze them as v1.1 characterization for `resolver`/`legacy` until those packages are deleted. Create **Go-native v6 goldens** from the new engine over the same contract personas (`cmd/contract` personas), with `-update` regeneration and hand-verified anchors from the §2.36 tables.
- Unit tests to rewrite: `resolver_test.go:124` (prediction_based_output), `metrics_test.go` regularity cases, `enums/logic_test.go:139-148` (RequiresUserInput set), `parity_test.go` (PHP parity of CycleWarning).

**Evolve in place or rebuild?** I recommend a **new pure package `internal/cycle/engine`**, with the old `resolver` kept frozen until its consumers move. Reasons:
1. `resolver`/`metrics`/`view` are a PHP port pinned by Laravel goldens, PHP-generated enums (`cmd/enumgen`) and parity tests. v6 changes semantics in about 15 places plus the output shape, so editing in place breaks all of that at once, and about 12 consumers (`home`, `fertility`, `challenges`, `checkups`, `teen`, `todo`, `healthlog/prefs`, `companion/shared`, `routes_shop_checklist`, `analysis`, `insights`, `profile/cyclesettings`) would change in the same commit.
2. v6 needs a different input model (LH/BBT/mucus series, skips, contraception, mode, age, tz) and per-user precomputation: BBT shifts per cycle, luteal median, 12-month regularity, flags. These belong in a `Prepare` step, not inside every per-day resolve.
3. A clean package can take `Params` as a value (a clinicalconfig snapshot resolved by the service), use int day numbers internally, and own hand-written v6 enums. That removes the global-state impurity and the PHP enum coupling.
4. Shadow-diff testing becomes possible: run old and new on the same personas, and assert the expected deltas only (O+1, warnings, the new fields).

Proposed API:
```go
type Params struct{ FertileBefore, HardCap, WarnExtra, WarnMin, MaxLate, LutealDefault int; BBTStrict bool; FIGO …; Version string }
type Input  struct{ Periods []Period; Skips []civildate.Date; Declared Declared; BirthDate civildate.Date;
                    Mode Mode; Hormonal bool; Postpartum *Postpartum; LH []LHTest; BBT []BBTReading; PositiveTest *civildate.Date; ManualLengths bool }
func Prepare(in Input, p Params) *State          // O(H log H + R): sort, cycles, medians, BBT shifts, luteal, regularity, flags
func (s *State) Resolve(target, today civildate.Date) Status   // O(log H) anchor binary search, no allocation besides output slices
func (s *State) Range(from, to, today civildate.Date) []Status // one pass per cycle, for /cycle/month and charts
func (s *State) FutureCycles(today civildate.Date) []FutureCycle
func TodayUI(st Status, s *State) UIHints                      // §2.43
```
Internally the dates become `int` epoch days; `civildate` is used only at the edges.

---

## 5. Proposed task breakdown (web + backend only; no Android, no production cutover)

### A. Pure library (`backend-go/internal/cycle/engine`)

**ENG-01 Engine skeleton, params, enums, metrics (§2.4–2.10, §2.27 variability)**
- Package, `Input`/`Params`/`State`/`Status` types, epoch-day helpers, hand-written v6 enums: main_phase, subphase, ovulation_source, regularity, 20 warnings + `contradictory_anchor_data` (pending decision 2), 14 reasons, resolution, data_quality, confidence.
- `Prepare`: real cycles (skips excluded), valid lengths, median half-up, effective values with the declared → default fallback and manual override, variability.
- `Params` built by a service helper from clinicalconfig. The engine never calls clinicalconfig.
- touches: `internal/cycle/engine/*` (new), `internal/clinicalconfig/registry.go` (new keys `engine.luteal_default`, `engine.bbt`, retire `fertile_window.after`).
- deps: decisions 1, 2, 9, 20.
- Tests: §2.9 medians ([28,29]→29, [27,28]→28, 3-value, 1-value, declared, default), skip excluded without an outlier warning, outlier flags, variability null/value.

**ENG-02 Core resolve (§2.11–2.26, §2.30–2.31, §2.34–2.36)**
- Anchor by binary search, end model, caps, priority, menstrual tiers, period_expected tiers, suggest_pregnancy_test, follicular split, fertile O−5..O, luteal with the v6 priority order, fertility_level, days_* fields, resolution_source, is_predicted.
- deps: ENG-01, decisions 11, 12.
- Tests: §2.36 tables 21/28/35 verbatim; §2.35 example (full JSON); days_late 7/8/14/15; open period days 6/9/13/20; empty fertile window; very short luteal range; O+1 never fertile; menstrual overrides fertile; Jalali boundaries (Esfand 1402 leap → Farvardin, 31→30-day months Shahrivar→Mehr) as Gregorian arithmetic checks; suggest_pregnancy_test matrix.

**ENG-03 Quality layer (§2.27–2.29, §2.32–2.33)**
- 12-month FIGO regularity by age, clinical_flags incl. teen, data_quality, confidence (with age ≥ 45 cap, k ≥ 2 cap, hormonal/postpartum low, bbt alternative), reasons, warnings dedup, requires_user_input.
- deps: ENG-02, decisions 6–9, 18, 19.
- Tests: the regularity/flags cases in §2.27 above; age 46 → medium + perimenopause; age 16 → teen ranges; each of the 14 reasons triggered once; requires_user_input exactly for the 6 warnings.

**ENG-04 Ovulation sources (§2.16 LH/BBT next, §2.17)**
- Move the 3-over-6 detector into `engine/bbt` (strict/standard via Params, ≥ 10 readings) and make `internal/fertility/bbt` call it. LH first positive + 1, conflict > 2 days, effective_luteal_length (median of BBT luteal lengths in 10..16, half-up, else 14), next = O + luteal.
- touches: `internal/cycle/engine/bbt`, `internal/fertility/bbt/bbt.go` (delegate).
- deps: ENG-02, decision 16.
- Tests: the BBT/LH cases in §2.17 above, strict vs standard, 9 vs 10 readings, conflict at 2 vs 3 days, luteal [12,13]→13 and [9,17]→14, BBT-anchored next.

**ENG-05 Future cycles, stale anchor, skipped cycle (§2.16 chain, §2.39.8, §2.41)**
- `FutureCycles` k=1..6 (shape per decision 5), future-target resolution (decisions 3–4), stale onboarding anchor n = ⌊diff/L⌋, skip semantics (decision 15).
- deps: ENG-03.
- Tests: chain dates L=29; k ≥ 2 ≤ medium; late user, target tomorrow; stale diff 42/43/90; skip on the expected day → next + L, warning, excluded from regularity.

**ENG-06 Limiting conditions (§2.42)**
- Hormonal (decision 13), pregnancy (suspended status), postpartum/breastfeeding (decision 14), positive test prompt, age caps, teen.
- deps: ENG-03.
- Tests: §6.1 pill, postpartum without a period, age 46, age 16, positive test → `prompt_pregnancy_mode`.

**ENG-07 Today-UI hints (§2.43)** (if decision 17 = engine)
- Pure `TodayUI(status)`: ring action, single banner by priority, prediction-card ranges.
- deps: ENG-05, ENG-06.
- Tests: the boundary list in §2.43 above.

**ENG-08 Benchmarks, invariants, v6 goldens**
- `BenchmarkPrepare`, `BenchmarkResolve`, `BenchmarkRange31`, with a target of 0 extra allocations per Resolve beyond the output.
- Property sweep over random histories × 400 days. Invariants: cycle_day ≥ 1 when a start exists; never fertile on a menstrual day; O+1 never fertile; every start/end has a source; fertility unknown iff (period_expected | unknown | suppressed); dto null rules.
- Go-native v6 goldens over the contract personas with `-update`.
- deps: ENG-02..06.

### B. Service / API

**SVC-01 Engine input loader + params + cache key.** Build `engine.Input` from cycle_histories (onboarding rows as declared), skips, contraception method → hormonal, mode/life_mode, birthday, postpartum/breastfeeding, LH/BBT from `fertility_logs`/`daily_health_logs`, and the user tz → `today`. Freeze `Params` per request from clinicalconfig. Key the cache on all inputs plus `Params.Version`. Touches `internal/cycle/service`. Deps: ENG-01, plus the schema tasks from the other report (skips table, tz, breastfeeding, onboarding row fix). Tests: integration (`make test-int PKG=./internal/cycle/...`), including the 00:30 Tehran on a UTC clock.

**SVC-02 `cycle_view` v6 payload.** Emit the §2.35 shape (with `future_cycles`, `clinical_flags`, `ovulation_source`, `suggest_pregnancy_test`, `today_ui`). Keep the legacy keys temporarily, rebuilt from the v6 status (`predictions` from the display window, O−5..O; a single confidence), behind a deprecation note. Update OpenAPI and the contract allowlist. Deps: SVC-01, ENG-05..07. Tests: contract case + v6 goldens.

**SVC-03 Skip endpoint + recompute.** `POST/DELETE /api/v1/cycle/skips` («پریود نداشتم»), validation (only when days_late ≥ 0), cache invalidation, past-edit full rebuild (§2.39.9). Deps: SVC-01, ENG-05. Tests: integration, edit a period 3 months back → medians and regularity change, cache busted.

**SVC-04 `/cycle/month` + calendar on `State.Range`.** Replace the per-day legacy phase/fertile flags (O+1 fix), add predicted days from `future_cycles`. Deps: SVC-02. Tests: a month spanning cycle boundaries, O+1 not fertile, a 31-day Tir / 29-day Esfand.

**SVC-05 Daily card on v6.** Rebuild `view/daily_card` inputs from the v6 status/today_ui. Drop `EvaluateOpenPeriod`/`Predict`/legacy subphase. Fertility unknown on period_expected. Deps: SVC-02. Tests: the existing card cases, re-expressed.

**SVC-06 Fix the onboarding seed** (if not already in the other report's tasks). Store the LMP as declared (`is_estimated`, source `onboarding_declared`, **no end**) so it neither enters medians nor blocks §2.41. Include a data migration for existing rows. Touches `internal/profile/service.go:465-513`.

### C. Consumers

**CON-01 Backend consumers → engine.** Migrate `home/cycle_overview.go`, `fertility/{service,chart,insights}.go` (use `Range`/fertile window directly; drop up to 60 resolves per cycle), `challenges`, `checkups/engine`, `teen`, `todo`, `healthlog/prefs`, `companion/shared`, `http/routes_shop_checklist.go`, `profile/cyclesettings.go`, `cycle/insights`, and `analysis` (regularity and FIGO flags come from the engine; remove the duplicate `markOutlier` rule per decision 9). Deps: SVC-02. Tests: each package's existing int tests, plus regularity agreement between the analysis tab and cycle_view.

**CON-02 Frontend.**
- `entities/cycle/api/schema.ts`: v6 fields.
- `screens/home/model/hero-state.ts` → read `today_ui`; banners (overdue ≥ 8 + test button, stale anchor, end missing); regularity badge; future cycles in the calendar; hide the fertile UI when hormonal; «پریود نداشتم» action wired to SVC-03; the persistent "prediction is not contraception" notice.
- Light and dark, fa/en.
- Deps: SVC-02..05. Tests: unit tests for the mappers, visual QA.

**CON-03 Retire v1.1.** Delete `resolver`, `view` legacy blocks, `legacy` engine phase paths, PHP-generated cycle enums usage, and the Laravel cycle-sweep golden replays (keep the files only if prod Laravel still needs them as a reference). Deps: CON-01, CON-02.

### §6.1 cycle tests → task mapping
21/28/35 table → ENG-02 · open period to day 13 + after hard cap → ENG-02 · days_late 7/8/15 → ENG-02 · empty fertile window + short luteal → ENG-02 · onboarding without history, default without onboarding → ENG-02/ENG-03 · median of two with .5 → ENG-01 · Jalali 31/30 + leap Esfand + §2.35 example → ENG-02 · O+1 outside the window → ENG-02 (+ SVC-04 for the calendar) · LH + BBT + conflict → ENG-04 · onboarding anchor 90 days ago, skipped period, edit 3 months back → ENG-05/SVC-03 · pill, postpartum, age 46, age 16 → ENG-06 · 00:30 Tehran on a UTC server → SVC-01.
