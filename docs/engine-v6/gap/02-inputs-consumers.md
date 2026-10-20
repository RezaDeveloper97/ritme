# M9 Engine v6: gap analysis 02 — engine inputs and engine consumers

- **Scope:** the edges of the cycle engine. That means what feeds it (inputs, the logging bridge, storage rules, schema) and what reads it (backend packages, frontend, cache invalidation). The core rules of `docs/engine-v6/SPEC.md` §2.x belong to the other report. They are cited here only where an input or consumer depends on them.
- **Method:** read-only sweep of `backend-go/`, `frontend/src/`, `backend/database/migrations/`, `docs/engine-v6/SPEC.md` and `docs/go-migration/deviations.md`, on 2026-10-21, branch `stage` with a dirty working tree. Two helper sweeps covered the frontend consumers and the second group of backend consumers; their findings are merged into §5 and §6. No repository file was changed.
- **Line refs:** paths are relative to the repo root. `backend-go/` is shortened to `bg/`, and `bg/internal/` to `bgi/`. An item marked **unsure** was not verified end to end.

---

## 1. Summary

1. **Most v6 inputs are stored but not read.** The engine snapshot (`bgi/cycle/service/service.go:73-102`, `db/queries/cycle/engine.sql:26-36`) reads these:
   - `cycle_histories`
   - one `user_profiles` row: LMP, declared lengths, birthday, goal
   - `cycle_preferences.lengths_auto`
   - `user_life_profiles.life_mode`, used only as a copy flag
   - `daily_health_logs` for the requested range, symptom columns only

   Of the 15 inputs in §2.6, 5 are wired. Two more are half-wired:
   - `last_period_start_from_onboarding` shares a column with "latest logged start".
   - `birth_date` is read only by the legacy engine.

   Seven are stored but never reach the engine: LH, BBT, mucus, mode, hormonal contraception, the positive pregnancy test and the postpartum birth date. Three are not stored at all: skipped cycles, a breastfeeding flag, and device timezone (§2.3).
2. **The onboarding seed breaks §2.2, §2.9, §2.40 and §2.41.** `profile.syncOnboardingPeriodLog` (`bgi/profile/service.go:463-513`) writes the declared LMP into `cycle_histories` as a confirmed, non-estimated `user_profile_confirmed` row. Its end is assumed (`LMP + declared duration − 1`) and its `bleeding_length` equals the declared duration. The resolver reports it as `user_logged` (`bgi/cycle/resolver/resolver.go:385-397`), so the declared value feeds the medians and §2.41 can never fire. **Recommendation:** store the declared anchor in its own column, then migrate the rows (§4).
3. **§1.4 bridge: the trigger is right, the rest is wrong.**
   - Right: a flow intensity creates the period and spotting does not.
   - Rows created from flow logs are unconfirmed. Only the newest one is promoted (`bgi/cycle/model/logged_start.go:12-22`, D-55, still *proposed*). Older log-derived periods never count toward medians or regularity.
   - Flow logged in pregnancy creates periods and moves the LMP. There is no mode guard.
   - Deleting or clearing a log never shrinks or removes the period it created.
   - A back-dated insert leaves the next row's stored `cycle_length` stale.
4. **Today there are three engines plus local math.**
   - The legacy engine serves `/cycle/month` (the calendar), the `/home` sections and the `calculation` block. Its cycle day wraps, so it never reports late; its biological window runs O−5..O+1; its PMS window is ≥ L−6.
   - The v1.1 resolver serves `cycle_view`, `/fertility/*`, `/home/cycle-overview`, companion, teen and checkups.
   - `view.Predict` builds `cycle_view.predictions` with a window of O−5..O+1.
   - On top of these, analysis, fertility insights, the home health check, reminders and the frontend `schedule.ts` each compute their own regularity, ovulation source or windows.

   Counts: five regularity notions, three fertile-window definitions, two PMS windows, and two copies of the BBT>LH>calendar ovulation-source logic.
5. **Cache invalidation is content-addressed but incomplete.** The key (`bgi/cycle/service/service.go:191-215`) covers the profile row, all history rows and the logs in the requested range. It does not cover:
   - the clinical-config version (stale until the day rolls over)
   - LH (`fertility_logs` and v2 entries)
   - BBT outside the range
   - contraception
   - `pregnancy_profiles`
   - postpartum data
   - `life_mode`, except when the fertility-copy flag flips
   - the v2 pregnancy test

   Every v6 input added to the engine must also be added to the key. There are no analysis caches today, so §2.39.9 "rebuild analysis caches" is a no-op until someone adds one.
6. **Timezone:** "today" is Tehran everywhere. There are 241 `civildate.Today` / `InTehran` call sites in 125 files, and the frontend pins `Asia/Tehran` (`frontend/src/shared/i18n/request.ts:20`). Supporting the §2.3 device timezone needs a client header, a stored per-user zone for jobs, and a scoped rollout.
7. **Schema:**
   - New storage is needed for the declared anchor, skipped cycles and timezone.
   - Breastfeeding and menarche granularity each need either a decision or a column.
   - `user_profiles` lacks `UNIQUE(user_id)`: the insert at `db/queries/profile/profile.sql:37` is a plain INSERT, and readers use `ORDER BY id LIMIT 1`.
   - The existing indexes are enough for every query v6 needs.
8. **Proposed work:** 12 tasks (§9). Order:
   1. A decision and deviation task, which needs the user's OK.
   2. Schema.
   3. Input loader with a unified cache key.
   4. Seed replacement and the bridge, in parallel.
   5. Skipped-cycle endpoint and timezone.
   6. Consumer convergence, in three backend tasks.
   7. Frontend contract tasks.

   No Android work and no production cutover are included.

---

## 2. Inputs (v6 §2.6, §2.3, §2.27, §2.42)

| v6 input | Storage today | Writer(s) | Engine reads it? | Gap |
|---|---|---|---|---|
| `period_start` / `period_end` (actual) | `cycle_histories.period_start_date` / `period_end_date` (`bg/db/migrations/00001_baseline.sql:129-145`, UNIQUE(user_id, period_start_date)) | Period button: `bgi/cycle/periods/service.go:94-167` (Start), `:170-199` (End), `:202-247` (Store), `:250-281` (Update), `:284-301` (Destroy). Flow bridge: `bgi/healthlog/cyclehistory.go:83-149`. Onboarding seed: `bgi/profile/service.go:463-513` | **Yes.** `service.Load` → `HistoriesFromRows` (`bgi/cycle/service/adapters.go:48-55`). The resolver anchors on confirmed rows only (`resolver.go:385-397`) | Rows from the flow bridge are `is_confirmed=0` (`bg/db/queries/healthlog/side_effects.sql:30-34`); only the newest is promoted. `periods.Update` keeps the row's old `is_confirmed`/`source` (`periods/service.go:267-269`), so a user editing a log-derived period does not confirm it. The seed is stored as actual data (see §4) |
| `cycle_history` (cycle lengths) | `cycle_histories.cycle_length`: a stored, derived column | `periods.recomputeCycleLengths` (`periods/service.go:322-343`); the bridge sets it only at insert (`cyclehistory.go:124-132`) | **Partly.** `metrics.Calculate` uses it for confirmed rows. The KPI package reads the stored column (see §5) | It goes stale: a back-dated bridge insert between two periods never recomputes the next row (`cyclehistory.go:133-147`). v6 effective lengths should come from start dates (§2.9) and treat the stored column as a cache, or recompute it on every write |
| `bleeding_history` | `cycle_histories.bleeding_length` + `period_end_date`; `daily_health_logs.bleeding_intensity` (`00001_baseline.sql:151`) | same as above | **Yes** (period-length medians) | The seed writes `bleeding_length` = the declared duration (`profile/service.go:483-490`), so a declared value enters `effective_period_length` (§2.9 forbids assumed ends) |
| `declared_cycle_length` / `declared_period_length` | `user_profiles.cycle_duration` / `period_duration`; `cycle_preferences.lengths_auto` (00013) | `profile.Save` (`bgi/profile/service.go:61-135`), onboarding `UpdateStep` (`bgi/profile/onboarding.go:525-613`), cycle settings | **Yes** (`adapters.go:58-76`; `LengthsManual` at `service.go:105-112`) | OK. Also duplicated into the seed row as if actual |
| `last_period_start_from_onboarding` | **No separate column.** `user_profiles.last_period_start` holds both the declared LMP and the "latest actual start" | `profile.Save`; bridge `updateProfileLMP` (`cyclehistory.go:212-224`); `periods.reanchor` / `setLastPeriodStart` (`periods/service.go:346-373`) | **Yes, but it's ambiguous.** The legacy engine needs it (`bgi/cycle/legacy/engine.go:90-98`). The resolver uses it as the `onboarding_declared` anchor | Every logged start overwrites it, and `reanchor` sets it to NULL when the last period is deleted, so the declared value is lost. §2.41 cannot tell a declared anchor from a logged one. `home.HasCycleData` means "LMP is set" (`bgi/home/context.go:151-153`), so deleting all periods also hides the home cycle sections |
| `birth_date` | `user_profiles.birthday` | `profile.Save` | **Legacy only** (`legacy/engine.go:272-278`, `ageFactor`). The v1.1 resolver never reads it | §2.27 FIGO thresholds by age and the §2.42 rules for age ≥45 and <18 need it inside the engine. Teen users may have only `teen_profiles.age_band` (`10_12`/`13_15`/`16_17`, `bg/db/migrations/00029_teen.sql:12-28`): a band, not a date |
| `lh_tests[date, result]` | `fertility_logs.lh_test` (00004; `negative`/`faint`/`positive`, `bgi/fertility/day.go:20-21`) **and** `health_log_entries` `measurements.lh_test` (`bgi/healthlog/taxonomy/taxonomy.go:250`) | `fertility.Save` (`bgi/fertility/service.go:93-153`); logs v2 `SaveDay` | **No.** The engine model has no LH field (`bgi/cycle/model/model.go:10-54`). Only fertility insights and analysis TTC read it, through `ListLHTests`, which unions both stores (`bg/db/queries/fertility/insights.sql`) | Two stores. Not in the snapshot or the cache key. `fertility.Save` does not mark a recalculation. §1.4 says LH is an engine input |
| `bbt[date, value]` | `daily_health_logs.basal_body_temperature` (`00001_baseline.sql:196`); v2 `measurements.bbt` (`taxonomy.go:249`), mirrored both ways (`taxonomy/legacy.go`) | health-log store / v2 `SaveDay` | **No.** `DailyLogFromRow` drops BBT (`adapters.go:80-98`) | The rule needs every reading in a cycle (≥10), but `DayJSON` loads logs for `date..date` only (`service.go:226-243`). The existing rule in `bgi/fertility/bbt/bbt.go:17-25,140-173` differs from v6 in two ways: `MinRise` (+0.20 °C) applies to the **first** high reading, where v6 wants the **third**, and there is no ≥10-readings minimum |
| `cervical_mucus[date, type]` | Three forms: `fertility_logs.cervical_mucus` (`dry`/`sticky`/`creamy`/`egg_white`), `daily_health_logs.discharge_texture` (`00001_baseline.sql:203`), v2 `discharge.consistency` (`taxonomy.go:206`) | fertility `Save`, health-log store, v2 | **No** | **Unsure** whether v6 needs it in the engine. §2.6 lists it, but no rule in §2.16-§2.34 reads it (only the §4.1 TTC analysis). Recommendation: leave it out of the engine (YAGNI) unless the core report finds a rule |
| `mode` | `user_life_profiles.life_mode` (00014); `pregnancy_profiles.pregnancy_mode`/`cycle_mode`; `user_profiles.user_goal`. Resolved by `enums.ResolveLifeMode` | `UpdateLifeStage` / `SwitchLifeMode` (`onboarding.go:686-763`). Pregnancy `Activate` / `Deactivate` / `Onboarding` (`bgi/pregnancy/handlers.go:115-212`) write **only** `pregnancy_profiles`. Postpartum `Activate` (`bgi/postpartum/service.go:113-145`) | **Only as a copy flag.** `NoFertilityCopy` comes from the stored `life_mode` (`service.go:105-112`). `engine.sql:26-36` does not join `pregnancy_profiles`, so an active pregnancy is invisible to the engine | §2.42 needs the effective mode inside the engine: pregnancy stops it, postpartum needs 2 actual periods, and `absent` (§2.27) is suppressed in pregnancy and postpartum |
| `hormonal_contraception` | `contraception_methods.method` (00017: `combined_pill`, `progestin_pill`, `copper_iud`, `hormonal_iud`, `injection`, `implant`, `condom`, `other`; UNIQUE user); `user_life_profiles.medications` (`contraceptive_pill`, `iud`, `hormonal_medication`, `bgi/enums/life_stage.go:156`); `track_contraception` | contraception module; onboarding conditions step (`onboarding.go:525-613`) | **No.** No `IsHormonal` helper exists (`bgi/contraception/pack.go:17-27`) | A derivation rule is needed. Hormonal: combined pill, progestin pill, hormonal IUD, injection, implant. Not hormonal: copper IUD, condom, other. The onboarding `iud` value cannot tell copper from hormonal, so that needs a **product decision**. §2.40 forbids showing the fertile window while this is true |
| `breastfeeding` | **No flag.** It can only be derived from `baby_feeds.type` `breast`/`pump` (00033) | baby log | **No** | Needs either an explicit flag (for example on `postpartum_profiles`) or a derivation rule. Needs a **decision** |
| `skipped_cycles` | **Not stored** | — | **No** | §2.8 and §2.39.8 need storage, an endpoint ("I had no period"), exclusion from medians and regularity, and the `skipped_cycle_recorded` warning |
| `positive_pregnancy_test` | v2 `measurements.pregnancy_test` (`negative`/`positive`, `taxonomy.go:251`), **not** mirrored to `daily_health_logs` | logs v2 `SaveDay` | **No** | §2.42 (prompt to change mode) and §2.22 (`suggest_pregnancy_test`) need the latest test result in the current cycle |
| postpartum `birth_date` (for §2.42) | `postpartum_profiles.birth_date` (00030) | postpartum | **No** | Needed to count "2 actual periods after delivery". Postpartum users also cannot log a returning period through `flow`, because `flow` modes are cycle-type plus pregnancy, not postpartum (`taxonomy.go:163`). That path needs a **decision** |
| menarche (§2.27 teen flag: "age <18 or ≤3 years since first period") | `teen_profiles.menarche` buckets `not_yet`/`under_1y`/`over_1y` (00029) | teen onboarding | **No** | The "≤3 years" rule cannot be computed from buckets. Either ask for the menarche year or apply only "age <18". Needs a **decision** |
| device timezone (§2.3) | **Not stored.** Today is `civildate.InTehran(now)` (`bgi/cycle/service/handlers.go:50-54`, `bgi/platform/civildate/civildate.go:56-59`); the frontend pins `Asia/Tehran` | — | **No** | The day boundary is wrong for users outside Iran. See IC-08 |

---

## 3. §1.4 bridge findings (logging → period)

The v6 rule is that logging a flow intensity starts a period and logging spotting does not. LH and BBT are engine inputs. Bleeding in pregnancy becomes an alert card ("share with your doctor"). All of it runs through `checkAndUpdatePeriodStart` (`bgi/healthlog/cyclehistory.go:83-149`). The legacy path reaches it via `service.store` (`bgi/healthlog/service.go:106-137`). The v2 path reaches it through `SaveDay` → `writeBackLegacy` → `store` (`bgi/healthlog/entries.go:180-246`), and `PUT /fertility/days/{date}` uses the same path (D-55).

| # | Finding | Evidence | v6 rule | Severity |
|---|---|---|---|---|
| B1 | **OK:** only `bleeding_intensity` (v2 `flow`: light/medium/heavy/very_heavy) triggers the bridge. Spotting is a separate boolean and never does. The menopause `presence` param has no legacy column, so it never creates a period | `cyclehistory.go:83-86`; `taxonomy/legacy.go:84-90`; `taxonomy.go:163-167`; `taxonomy/menopause.go:32` | §1.4 | — |
| B2 | Bridge rows are unconfirmed. Only the newest unconfirmed `user_logged` row is promoted, so older log-derived periods stay out of anchors, medians and regularity. A user who only ever logs flow keeps one effective record forever, falls back to declared lengths and gets `insufficient_data` regularity | `side_effects.sql:30-34` (`is_confirmed` 0); `logged_start.go:12-22`; D-55 text: "older unconfirmed rows stay out of anchors and medians" | §2.2 and §2.9: user-logged periods are actual data | **High** |
| B3 | **No mode guard.** In pregnancy, `flow` is offered (modes cycle-type plus pregnancy), so a bleeding log creates a `cycle_histories` row and moves `user_profiles.last_period_start`. **Unsure:** menopause is in the cycle-type set too, so a menopause `flow` log would also create a period if the menopause log screen shows `flow` (not verified in the UI) | `service.go:106-137`, `cyclehistory.go:83-149`, `taxonomy.go:55-58,163` | §1.4: pregnancy bleeding becomes an alert, not a period. §2.42: the cycle engine is stopped in pregnancy | **High** |
| B4 | **Deleting or clearing a log never rebuilds the period.** `Delete` and `DeleteDay` don't touch `cycle_histories`, and clearing `flow` on a day leaves the period's end as it was | `healthlog/service.go:250-257`; `entries.go:249-266` | §2.39.9: editing the past rebuilds the history | **High** |
| B5 | A back-dated insert between two periods records `cycle_length` from the previous period but never recomputes the **next** row's stored `cycle_length`. `periods.recomputeCycleLengths` exists but the bridge doesn't call it | `cyclehistory.go:124-147` vs `periods/service.go:322-343` | §2.39.9 | Medium |
| B6 | A back-dated new row is closed as a 1-day period (`bleeding_length` 1); later days extend it. That's fine, but the rule in `rules.go:23` flags it as `short_period` | `cyclehistory.go:139-147`; `bgi/cycle/periods/rules.go:18-29` | §2.8 (period length <2 is invalid for calculation) | Low. The flag goes away once the period extends |
| B7 | Two writers record the same fact with different provenance. The button (`periods.Start`/`Store`) writes confirmed `user_logged` rows and deletes estimates; the bridge writes unconfirmed `user_logged` rows. Nothing marks a row as "derived from logs" (the `source` default is `user_logged`), so a rebuild for B4 can't safely tell which rows it may delete | `periods/service.go:119-128,217-222`; `00001_baseline.sql:136-138` | — | Medium (it blocks B4) |
| B8 | Postpartum has no way to log a returning period through the log sheet. `flow` is not offered in postpartum and `lochia_amount` doesn't map to a period | `taxonomy.go:163,173` | §2.42: postpartum prediction resumes after 2 actual periods | Medium (needs a decision) |
| B9 | LH and BBT reach only `/fertility/*` and `/analysis/*`, never the engine (see §2) | `adapters.go:80-98`; `model.go:10-54` | §1.4 + §2.17 | **High** (input for the core engine) |
| B10 | A spotting-day warning uses its own cycle math: profile LMP + declared `cycle_duration`, not the engine's effective values | `cyclehistory.go:31-46` (`IsLikelyLutealSpotting`) | single source | Low |

---

## 4. Storage-rule findings (§2.2, §2.9, §2.39-§2.41)

| # | Finding | Evidence | Rule broken |
|---|---|---|---|
| S1 | **Onboarding seed stored as actual.** It is a `cycle_histories` row with `source=user_profile_confirmed`, `is_confirmed=1`, `is_estimated=0`, `period_end_date = LMP + declared − 1` (an assumed end stored as actual, unless that end is still in the future) and `bleeding_length = declared`. It is re-synced while it is the user's only row | `bgi/profile/service.go:463-513`. Laravel origin: `backend/database/migrations/2026_08_02_000001_promote_onboarding_estimates_to_logged_periods.php:6-43`, which promoted the old estimates "so the calendar paints those days". `backend/app/Enums/DataSource.php:45-48` counts `USER_PROFILE_CONFIRMED` as actual | §2.2 (assumed end never stored as a real end; onboarding data keeps its own source), §2.6 (declared ≠ actual log), §2.9 (assumed end never feeds medians), §2.40 |
| S2 | The engine reports the seed as `user_logged`: confirmed rows give `user_logged`, while `onboarding_declared` is reached only through `is_estimated` rows or the profile LMP. With a seed present, `resolution_source` is never `onboarding_based`, `onboarding_data_used` never fires, and §2.41 (stale anchor → `predicted_reference`) can never fire. A user who never logs again therefore keeps "high/medium" confidence based on a declared date | `resolver.go:385-397` | §2.10, §2.28-§2.30, §2.41 |
| S3 | `last_period_start` serves as both the declared anchor and a cache of the latest start. Logs overwrite it, and `reanchor` sets it to NULL when all periods are deleted | `cyclehistory.go:212-224`; `periods/service.go:346-373` | §2.6 (a separate declared input) |
| S4 | **OK:** no predicted start is stored. `cycle_calculations` was dropped (`backend/database/migrations/2026_09_19_000002_drop_cycle_calculations_table.php`; it is absent from goose). `future_cycles` are not stored. `periods.Store` deletes leftover `is_estimated` rows (`periods/service.go:236-238`). The resolver's `is_estimated` path is effectively dead | — | §2.40 holds |
| S5 | Editing a period through `PUT /cycle/period/{period}` (`bgi/http/routes_cycle.go:45`) keeps the row's old `source` and `is_confirmed`. An edited seed stays `user_profile_confirmed`, and an edited bridge row stays unconfirmed | `periods/service.go:267-269` | §2.39.2: a real period_start opens a confirmed cycle |
| S6 | The stored `data_quality_flags` (`long_period` >10, `short_period` ==1, `irregular_cycle` outside 21..45) match the §2.7 and §2.8 bounds, but they are written only by the period editor (`reconcile`), never by the bridge. **Unsure** whether any v6 consumer reads them | `periods/rules.go:9-30`; `periods/service.go:305-320` | — |

**Keep or replace for the onboarding seed: replace it.** This needs a deviation entry and the user's OK, because it changes `/cycle/*` and `/home` output for users who only have the seed.

1. **New column `user_profiles.onboarding_last_period_start` (DATE NULL).** This is the declared anchor. `profile.Save` writes it and nothing else overwrites it. `last_period_start` keeps meaning "latest actual start" for legacy readers until they are migrated.
2. **Data migration** (goose plus a mirrored Laravel migration). For each `user_profile_confirmed` row:
   - **Untouched** (`created_at = updated_at`, or its start/end still equal the profile LMP plus declared duration): copy the start into `onboarding_last_period_start` and **delete** the row.
   - **Edited by the user** (it differs from what a re-sync would write, or `updated_at > created_at` with no profile change at that time). **Unsure** how reliably this can be detected; the matching rule is the safer one. Convert it to a confirmed `user_logged` row. The user has stated it is a real period.
   - The `down` direction restores the rows.
3. **Remove** `syncOnboardingPeriodLog` and the `InsertOnboardingCycleHistory` / `UpdateOnboardingCycleHistory` queries.
4. **The resolver anchors on the new column** when no actual rows exist. This gives `onboarding_declared` and `onboarding_based` output, the §2.41 stale-anchor rule, and `confidence`/`data_quality` per §2.28-§2.29.
5. **The calendar's onboarding period** becomes an *assumed* range: a display marker computed by the engine, not a stored row. This is the visible product change that reverses the August 2026 decision, so it needs the user's OK.

Alternative, if the user wants the seed kept: keep the row, but store it as `is_estimated=1`, `source=onboarding_estimate`, with no `period_end_date` and no `bleeding_length`. The engine then treats it as declared. That is cheaper, but it keeps declared data inside `period_logs`, which §2.2 explicitly separates.

---

## 5. Backend consumers

"Shared engine?" means one of three things: the code reads the snapshot's v6 output, calls the resolver itself, or computes its own math.

| Package / file:line | Computes | Uses the shared engine? | Divergence risk |
|---|---|---|---|
| `bgi/cycle/service/service.go:151-163` `Snapshot.Day` → `GET /cycle/today`, `/cycle/date/{d}` | `calculation` (legacy, D-30 display window) + `cycle_view` (`view.Build`: resolver + `view.Predict`) | Mixed: legacy + resolver + `Predict` | `cycle_view.predictions` uses O−5..O+1 (`bgi/cycle/view/services.go:14-55`; `bgi/clinicalconfig/registry.go:89-90` `engine.fertile_window` after=1) with its own confidence (`view/builder.go:116-185`), while the top level uses O−5..O. That gives two windows in one response. v6 §2.18 says O−5..O |
| `service.go:166-182` `Month` → `GET /cycle/month` (the calendar) | per-day legacy `CalculateForDate` | **Legacy only**. D-30: "`/cycle/month`, `/home` and the calendar keep the legacy engine" | The cycle day wraps modulo L, so the calendar never shows lateness (`legacy/engine.go:186-219`). Ovulation is `max(L−13, 7)` (`:221-224`), the window is O−5..O+1, and PMS is ≥ L−6 (`:250-256`). All three differ from v6 |
| `bgi/home/context.go:158-233` + `sections.go:88-235` → `GET /home` | `week_calendar` (`is_ovulation` phase), `next_period`, `cycle_prediction`, challenge cycle day (`:403`) | **Legacy** | `next_period` uses `daysUntil := max(0, cycleLength−cycleDay+1)` (`sections.go:145`), so it never shows late. `cycle_prediction` uses `fertileStart, fertileEnd := ovulationOcc-5, ovulationOcc+1` (`sections.go:205`). It doesn't honour contraception (§2.40) |
| `bgi/home/sections.go:922-1036` health-check items | "normal" ranges of 21-35 days for cycles and 2-8 for periods | own constants | A third standard. v6 says the user-facing criterion is the §2.27 `clinical_flags`/`cycle_regularity` (FIGO), and §2.40 forbids using 21-45 as "normal" |
| `bgi/home/cycle_overview.go:49-70` → `GET /home/cycle-overview` | predicted period, PMS = next−3..next−1, regularity | resolver + `metrics` | PMS matches v6 §2.43. Regularity is `metrics.RegularityStatus`, not the v6 FIGO-by-age rule with `insufficient_data` |
| `bgi/home/support.go:59-83` digest | averages, shortest and longest | own, over **all** history rows | Includes unconfirmed and seed rows that the resolver ignores |
| `bgi/fertility/service.go:40-87` → `/fertility/*` | resolver, called directly and **uncached**, with `LengthsManual` | resolver | Consistent, but `chart.go:94-110` and `insights.go:283-304` call `resolve` once per day in loops, so a request runs many resolves |
| `bgi/fertility/insights.go:308-334` `historyRow` | per-cycle ovulation, BBT shift −1, else first LH +1, else estimate | **own** | A copy of the §2.17 source logic outside the engine (the second copy is in `analysis/ttc.go`). There is no `lh_bbt_conflict` rule |
| `bgi/fertility/bbt/bbt.go:17-25,140-173` | 3-over-6 shift | own | Rule differs from v6 (`MinRise` on the first reading, no ≥10 minimum). It should move into, or be shared with, the engine |
| `bgi/analysis/input.go:104-145` | `typical()` median over the window's cycles; layout O = L−13, fertile O−5..O | own | Its median runs over the analysis window, not the last 3 valid cycles (§2.9). The layout matches v6 |
| `bgi/analysis/cycle.go:18,117-162` | FIGO regularity `regular`/`irregular`/`not_enough_data`, frequency, prolonged | **own** | Enum mismatch (v6: `insufficient_data`), and it is a second FIGO implementation. **Unsure** whether its age thresholds match §2.27 |
| `bgi/analysis/ttc.go:13-35,286-350` | ovulation source BBT > LH > estimate, luteal length | **own** | A duplicate of fertility insights. v6 §2.16 needs the luteal median (10..16) inside the engine |
| `bgi/analysis/handlers.go:72-110` | loads through `cycleservice.Load` | snapshot | No cache, so it is always fresh. A v6 snapshot change propagates to it automatically |
| `bgi/healthrecord/service.go:452-527` | `analysis.BuildCycle`; `last_period_start` from any history row | analysis | Counts the seed as a period, and inherits the analysis regularity |
| `bgi/companion/shared/shared.go:105-138`, `mode.go:15-18` | resolver view; `MaxLateDays = 14` | resolver | The cap is hard-coded instead of coming from the resolver or clinical config |
| `bgi/teen/service.go:247-285` | resolver + `WeekBucket` roll-forward | resolver + own | Own roll-forward by design, to hide lateness. Keep it, but feed it the v6 `predicted_next_period_start` |
| `bgi/checkups/engine/cycle.go:25-75`; `bgi/checkups/service.go:58-71` | resolver start and length + own `Start + k·L` projection | resolver | Builds its profile **without `LengthsManual`**, so manual-length users get a different effective length than `cycle_view` |
| `bgi/profile/cyclesettings.go:108-135,155-183,193-230` | `metrics.Calculate` over its own adapter; `reminders.PMSCycleDay(cycleLen)` | metrics (own adapter) | Its adapter skips `PromoteLoggedStart`, so the newest flow-log period is missing from the settings screen. The PMS day comes from the reminders helper, not the resolver |
| `bgi/healthlog/cyclehistory.go:31-46` | luteal-spotting heuristic | own | Uses the declared length and LMP |
| `bg/db/queries/conditions/conditions.sql:90` | reads `cycle_duration`/`period_duration` directly | declared values | Ignores effective lengths. See the group-B notes below |

**Second group of consumers** (helper sweep; the main items were spot-checked against the code).

| Package / file:line | Computes | Uses the shared engine? | Divergence risk |
|---|---|---|---|
| `bgi/messages/source.go:135-160` + `manager/manager.go:202-218` → `GET /messages/daily` | phase, subphase, cycle day, ovulation day, fertile and PMS flags. These feed the base message, TTC `fertility_info`, the correlation rules and `context_info` | **Legacy, through its own loader.** It builds `legacy.New` over raw rows (`source.go:148-151`) and skips `PromoteLoggedStart` and `LengthsManual`. D-55's consumer list leaves messages out | The cycle day wraps, so a late user is told it's "day N of a new cycle". PMS is ≥ L−6 (7 days); v6 says next−3..−1. Phase names are legacy. No contraception, postpartum or age gates. On the same day it can disagree with the home hero (resolver) and the home smart tip (snapshot legacy engine, `home/context.go:158`) |
| `bgi/messages/v2.go:56-77` → `/messages/selected` and the admin rule preview (`bgi/admin/messages/rules.go:609`) | rule facts: `cycle_day`, `phase`, `subphase`, `days_to_period`, `is_pms_window`, `is_fertile_window`, `pregnancy_week` | legacy without D-28's display window | O+1 counts as fertile and as "ovulation", which contradicts D-28. `days_to_period = max(L−day+1, 0)` is never late (`v2.go:66`). Teen and menopause users get fertile facts. **Bug:** `pregnancy_week = today.DiffDays(lmp)/7+1` (`v2.go:73`), and `DiffDays` returns `other − d` (`bgi/platform/civildate/civildate.go:103-107`), so the week is negative for every pregnant user. There is no `main_phase`, `days_late` or `data_quality` fact |
| `bgi/messages/conditionnudges/engine.go:146-171`, `rules.go:55-59,156-166` | the "current cycle" window for the heavy-pain and heavy-bleeding nudges | own | Anchors on the profile LMP or any history row, including unconfirmed ones. Uses its own 45-day staleness cut-off and the profile `cycle_duration` |
| `bgi/reminders/planner.go:36-54,87-99` | dates for the before-period, PMS (next−3) and fertile reminders | `view.Predict` | **No caller.** Outside the package only `PMSCycleDay` is used (`bgi/profile/cyclesettings.go:208`), and the package says no sender exists (`planner.go:4-7`). When a sender is built it must use the v6 output: today's planner rolls past lateness and uses the biological fertile start |
| `bgi/todo/service.go:274-292`, `suggest.go:41-71` | the "your period is N days away" offer and its task due date | resolver via `cycleservice.Load` ✓ | The mode gate uses the **raw stored** `life_mode` ("" allowed, `suggest.go:41`), not `enums.ResolveLifeMode`, so a pregnant user with no stored mode is not excluded. `todo_suggestion_events.ref_date` is the predicted start: when an edit moves the prediction, a dismissed offer comes back |
| `bgi/challenges/service.go:70-92` | today's `main_phase` for eligibility | resolver over its **own copy** of the loader (`:74-91`) | The math is right; the loader is duplicated |
| `bgi/http/routes_shop_checklist.go:65-86`, `bgi/shop/checklist/supply.go:27-43`, `service.go:246-295` | the supply-reminder day | resolver via `cycleservice.Load` ✓ | **Stored** as a `reminders` row and re-synced only on `GET`/`PUT /shop/supply-reminder`, so it goes stale after a period edit or a pregnancy. Gates on the raw stored mode, and needs at least one history row (todo doesn't) |
| `bgi/labs/context.go:45-50,74-88` | the "approximate phase today" sent to the AI prompt and stored in `lab_reports.interpretation` | **own**: profile LMP only, `day % L`, O = `max(1, L−14)` (`context.go:87`), one day off the engine | Computed for *today*, not the lab's `taken_on` (`labs/service.go:578`). Stored and never refreshed. Ignores `cycle_histories` |
| `bgi/conditions/pmdd.go:232-274`, `service.go:186-249`, `pbac.go:121-135` | PMDD cycles and windows; which period a PBAC day belongs to | **own**: the last 3 history rows with no `is_confirmed` filter (`db/queries/conditions/conditions.sql:79-90`), plus the profile `cycle_duration` | The back-dated 1-day rows from D-55 can split cycles. No skipped-cycle handling. Uses declared, not effective, length |
| `bgi/search/mine.go:44-54,240-245`, `sources.go:88-104` | insight window and peak cycle day | **own**: the newest row (any) or the LMP, with a 60-day staleness cut-off | Yet another staleness rule. v6 has one: more than 14 days late → `unknown` |
| `bgi/assistant/profile.go:15-56` → `healthrecord/service.go:452-525` → `analysis/cycle.go:95-164` | the cycle section sent to the LLM: medians, variation, FIGO regularity, last start, current day | analysis | `median_cycle` ≠ `effective_cycle_length`. Regularity is separate from the engine's. The last start comes from any row, including the seed |
| `bgi/metrics/metrics.go:297-360` | engine MAE and ±2-day share (these feed the OKRs) | **own**, as raw SQL through `s.db.QueryContext` (`metrics.go:307`), which breaks the sqlc-only rule (`backend-go/CLAUDE.md:79`) | It measures the stored `cycle_length` against the median of the previous stored values, over every non-estimated row **including unconfirmed ones**. The engine predicts from confirmed starts. The confidence and `data_quality` distributions that `SPEC.md:874` requires are missing (`metrics.go:302-304`). **Probably owned by the KPI/admin report** |
| `bgi/admin/users/users.go:183-191`, `admin/users/v2.go:377-402` | raw profile LMP and lengths only | none | SPEC §5.2 wants today's engine output in the detail view, and `data_quality` plus the main warning in the list. **Probably owned by the KPI/admin report** |
| admin config: `bgi/admin/content/challenges.go:20-21,124-128` (daily-challenge `cycle_day` 1-35, matched against home's wrapped legacy day, `home/sections.go:403-404`); `bgi/admin/checkups/rules.go:25,96-108` (1-45) | validation only | — | Once `cycle_day` stops wrapping in v6, a late user's day no longer matches day-keyed content. The 35-day cap is below the v6 maximum of 45 |
| No cycle math | `care/today.go`, `notifications/prefs.go`, `notify`, `analyticsevents`, `ivf/cycle.go:81-114` (the engine doesn't know about an active IVF cycle; **unsure** whether v6 needs it), menomessages, pregnancyalerts, postpartum | — | — |

**How many definitions exist.** PMS has three: legacy ≥ L−6 (7 days), the legacy sub-phase `pms_possible` (2 days, `bgi/cycle/resolver/phase_mapper.go:41-48`), and planner/resolver next−3..−1, which matches v6 §2.43. "The current cycle is stale" has four: nudges at 45 days, search at 60, labs at 4·L, and PMDD with no limit. v6 has one rule (§2.22).

**Two consumers store derived cycle values** and need a resync hook or to stop storing them: the shop supply `reminders` row and the lab interpretation phase. A third, `todo_suggestion_events.ref_date`, is acceptable as-is, but its behaviour should be documented.


---

## 6. Frontend consumers and contract-change impact

The frontend sweep was written up separately: see [`02b-frontend-consumers.md`](02b-frontend-consumers.md) (zod inventory, consumers, duplicated client math, §2.43 gap table, query invalidation, tests) and the second pass [`02c-frontend-impact.md`](02c-frontend-impact.md) (main risks and per-change impact).

---

## 7. Recompute and invalidation matrix

The cycle cache key (`service.go:191-215`) hashes the user profile row, all history rows, the range bounds, the range's `daily_health_logs`, the tips signature (with content only), and the `lengths_manual` / `no_fertility_copy.v2` flags. Around those it adds `calculation_version`, locale, `Today` and scope. TTL is 24 h and the schema is 3 (`bgi/cycle/cache/cache.go:38-42`). `calculation_version` is bumped by `profile.MarkRecalculated` (`bgi/profile/service.go:178-186`), which the periods service, the health-log store (only when an LMP is set, `bgi/healthlog/service.go:129-137`), `profile.Save` and `POST /cycle/recalculate` (`bgi/cycle/service/handlers.go:185-219`) all call.

| Write (endpoint / function) | Changes a key input? | Bumps `calculation_version`? | Fresh in v6? | Gap / action |
|---|---|---|---|---|
| Period start/end/store/update/destroy (`periods/service.go`) | yes (history rows, profile LMP) | yes | yes | — |
| Health-log save, v1 and v2 (`healthlog/service.go:106-137`, `entries.go:180-246`) | yes (history if the bridge acted; that day's log in that day's key) | only when `last_period_start` is set | partly | Once BBT is read across the cycle, a BBT log must change the key of **every** day in the cycle, but today only the logged date's key changes |
| Health-log delete (`service.go:250-257`, `entries.go:249-266`) | that day's log only | no | no | Add the period rebuild (B4). The key then follows from the history change |
| Fertility day save, LH/mucus (`fertility/service.go:93-153`) | **no** (`fertility_logs` is not loaded) | no | **no** | Add LH to the snapshot and the key |
| v2 `pregnancy_test` entry | **no** (v2 only; not mirrored to `daily_health_logs`) | no | **no** | Add to the snapshot and the key |
| `profile.Save`: LMP, declared lengths, birthday, goal (`profile/service.go:61-135`) | yes (profile row) | yes when cycle fields change | yes | Add the new declared-anchor column to the key (it is part of the row, so this happens automatically) |
| Cycle settings: `lengths_auto` (`PUT /profile/cycle-settings`, `bgi/profile/cyclesettings.go:316-322`) | yes (flag) | no (manual lengths go through `profile.Save`, which bumps it) | yes | — |
| `life_mode` switch (`onboarding.go:686-763`) | only when `AllowsFertilityContent` flips | no | **no** | Mode becomes an engine input, so put the effective mode in the key |
| Pregnancy activate/deactivate/onboarding (`pregnancy/handlers.go:115-212`) | **no** (only `pregnancy_profiles` is written) | no | **no** | Put the effective mode in the key |
| Postpartum activate (`postpartum/service.go:113-145`) | incidentally yes: `SyncPostpartumLegacyGoal` updates `user_profiles.updated_at` (`bg/db/queries/postpartum/*.sql:46-53`) | no | by accident | Put the postpartum birth date and mode in the key explicitly |
| Contraception method set/change | **no** | no | **no** | Put `hormonal_contraception` in the key |
| Baby feeds (if breastfeeding is derived from them) | **no** | no | **no** | Put the derived flag in the key, or store an explicit flag |
| Skipped cycle (new) | — | — | — | Put it in the key |
| Clinical-config publish (admin) | **no** (`clinicalconfig.Fingerprint` exists but only vitals use it) | no | stale until the day rolls over | Add the fingerprint of the `engine.*` keys to the key |
| Tips / recommendations content | yes (signature) | — | yes | — |
| Device timezone (new) | `Today` is part of the key | — | yes, if `Today` is user-local | — |
| **Derived values stored outside the cache:** shop supply `reminders` row (`bgi/shop/checklist/service.go:246-295`), lab interpretation phase (`bgi/labs/service.go:578-587`), `todo_suggestion_events.ref_date` (`bg/db/queries/todo/todo.sql:76-82`), `metric_snapshots` (hourly) | not keyed: these are separate rows | — | **no**, except metrics (hourly) | Shop: resync it from the period, health-log and mode writers, or compute it when read. Labs: compute the phase for `taken_on` from the engine when the interpretation is made, and store it labelled with the date it was computed for. Todo: acceptable; document it |
| Analysis caches (§2.39.9, §4.1) | n/a: analysis runs per request (`analysis/handlers.go:72-110`) | — | yes | If analysis is ever cached, key it on the same engine-input digest |

**Recommendation (IC-07).** Stay content-addressed: the snapshot already reads every input, so a missed `MarkRecalculated` can't serve stale data. Replace the ad-hoc flags with one `InputsDigest` over the whole v6 input set plus the `engine.*` clinical-config fingerprint, and bump the cache schema to 4. Keep `calculation_version` only as the client-visible "recalculated" signal. Frontend query invalidation after mutations is covered in §6.

---

## 8. Schema and index findings

| # | Finding | Evidence | Action |
|---|---|---|---|
| X1 | `user_profiles` has only a non-unique `KEY(user_id)`. The insert is a plain INSERT, and every reader uses `ORDER BY id LIMIT 1`. Some writers update `WHERE user_id = ?`, hitting every row (`SyncPostpartumLegacyGoal`); others update `WHERE id = ?` | `bg/db/migrations/00001_baseline.sql:675-696`; `bg/db/queries/profile/profile.sql:28,37`; `cycle/engine.sql:6,26-36`; `postpartum/*.sql:46-53` | Add `UNIQUE(user_id)` after a duplicate count on stage and beta. **Unsure** whether duplicates exist; the migration has to merge or fail loudly. Goose plus a Laravel mirror |
| X2 | `cycle_histories`: `UNIQUE(user_id, period_start_date)` is the only secondary index, and every engine and bridge query is user-prefixed and ordered by start date | `00001_baseline.sql:129-145`; `db/queries/cycle/*.sql`, `healthlog/side_effects.sql` | Enough. Per-user row counts are small, so no new index is needed |
| X3 | `health_log_entries`: `UNIQUE(user_id, log_date, category, param, item)` + `KEY(user_id, category, param, log_date)` | `bg/db/migrations/00020_health_log_entries.sql:41-44` | Enough for "all BBT, LH or pregnancy tests in a date range" |
| X4 | `fertility_logs`: UNIQUE(user, date); `daily_health_logs`: UNIQUE(user_id, log_date) | 00004; `00001_baseline.sql:213-215` | Enough |
| X5 | No storage for the declared anchor | — | `user_profiles.onboarding_last_period_start DATE NULL` (IC-02) |
| X6 | No storage for skipped cycles | — | New table `cycle_skips(id, user_id FK, expected_start DATE, created_at, updated_at, UNIQUE(user_id, expected_start))`. A separate table is better than a `cycle_histories` flag because a skipped cycle has no start date to fit the existing unique key (IC-02) |
| X7 | No timezone storage | grep for `timezone` columns comes up empty | `user_profiles.timezone VARCHAR(64) NULL` (IANA), needed for background jobs (IC-02, IC-08) |
| X8 | No breastfeeding flag | `baby_feeds` only (00033) | Decision: an explicit `postpartum_profiles.breastfeeding TINYINT NULL`, or derive "a breast or pump feed in the last N days". The explicit flag is simpler and privacy-neutral |
| X9 | No provenance for log-derived periods | `cycle_histories.source` default `user_logged` | Add a source value such as `bleeding_log` that the engine treats as actual (`user_logged` in output). It is varchar, so no DDL is needed, but the `DataSource` enum in Go and in Laravel's `DataSource.php` both need it. Needed for B4 rebuilds |
| X10 | Every schema change needs a goose migration **and** a mirrored Laravel migration until T-M2-27, and `make schema-diff` must stay green | `bg/CLAUDE.md:80-83` | Applies to IC-02 and IC-04 |

---

## 9. Proposed task breakdown

The IDs are local (`IC-nn`); the parent renumbers them into `T-M9-nn`. The fields follow `tasks/M8/*.md` front matter. All tasks are web and backend only. None includes Android work or the production cutover.

**Dependency order:** IC-01 → IC-02 → IC-03 → {IC-04, IC-05, IC-06, IC-07, IC-08} → IC-09a/b/c → IC-11+ (frontend). IC-10 has no dependencies and can run at any time. The core engine tasks from report 01 sit between IC-03 and IC-09, since the resolver consumes the new inputs.

### IC-01 — Decide the v6 input questions and write the deviation entries (needs the user's OK)
```
type: docs   depends_on: []   parallel_group: M9-IC-A
touches: [docs/go-migration/deviations.md, docs/engine-v6/]
model: opus (health logic) — main session asks the user
```
- Record the decisions that block the other tasks (marked **decision** in §2-§4):
  1. Onboarding seed: replace it with a declared anchor (recommended, §4) or keep it as an estimate. Decide how the calendar shows the declared onboarding period.
  2. D-55 revision: periods created from bleeding logs count as actual, with provenance `bleeding_log`. Deleting or clearing a log rebuilds the period. Pregnancy-mode `flow` raises an alert only, with no period row.
  3. Hormonal-contraception mapping from `contraception_methods.method`, and what to do with the onboarding `medications: iud` value, which doesn't say copper or hormonal.
  4. Breastfeeding source: an explicit flag or derived from `baby_feeds`.
  5. How postpartum users log their first periods (`flow` after lochia, or a "my period came back" action).
  6. Teen menarche: ask for the year, or apply only the "age <18" rule.
  7. Device timezone: header plus a stored column, and which packages switch in M9.
  8. Cervical mucus stays out of the engine (confirm with report 01).
  9. One "normal" criterion (§2.27) replaces home's 21-35 / 2-8.
- New deviation rows (status *proposed*) for 1, 2, 7 and 9. Revise D-30, because the calendar and `/home` leave the legacy engine.
- **Test plan:** n/a (review). Done when every decision has the user's answer recorded.

### IC-02 — Schema for the v6 engine inputs (goose plus mirrored Laravel migration)
```
type: backend   depends_on: [IC-01]   parallel_group: M9-IC-B
touches: [backend-go/db/migrations, backend/database/migrations, backend-go/db/queries/cycle, backend-go/internal/cycle/store, backend-go/internal/profile/store, backend-go/internal/enums]
verify: cd backend-go && make sqlc && make schema-diff && go vet ./... && make test-int PKG=./db/migrations/...
```
- `user_profiles.onboarding_last_period_start DATE NULL` and `user_profiles.timezone VARCHAR(64) NULL`.
- New table `cycle_skips(id, user_id FK cascade, expected_start DATE, created_at, updated_at, UNIQUE(user_id, expected_start))`.
- `postpartum_profiles.breastfeeding TINYINT(1) NULL`, only if IC-01 chooses an explicit flag.
- `DataSource` gains the value `bleeding_log`. This is enum code only; `source` is a varchar, so no DDL.
- No data moves here; the seed migration is IC-04.
- **Test plan:** migration integration test that runs up and down on the test stack; `make schema-diff` stays green.

### IC-03 — Engine input snapshot v6 (loader, adapters, one profile builder)
```
type: backend   depends_on: [IC-02]   parallel_group: M9-IC-B
touches: [backend-go/internal/cycle/model, backend-go/internal/cycle/service, backend-go/db/queries/cycle/engine.sql, backend-go/internal/contraception, backend-go/internal/checkups/service.go, backend-go/internal/profile/cyclesettings.go]
verify: cd backend-go && go vet ./... && go test ./internal/cycle/... && make test-int PKG=./internal/cycle/...
model: opus
```
- Add `model.Inputs` with: declared anchor, birth date (or teen age band), effective mode (`enums.ResolveLifeMode`, **including** `pregnancy_profiles`), hormonal contraception (`contraception.IsHormonal`), breastfeeding, postpartum birth date, skipped cycles, LH tests (both stores, reusing the `ListLHTests` union), BBT readings (both stores), latest pregnancy test, and the user-local `Today`.
- `service.Load` reads them:
  - The profile and mode flags come in one joined query.
  - The cycle-wide marker reads are bounded, for example from the start of the 7th newest period, or 400 days back.
  - `ListDailyLogsBetween` stays range-limited, because tips need only that range.
- Remove the per-package profile builders. `checkups/service.go:58-71` loses `LengthsManual` today, and `cyclesettings.go:155-183` skips `PromoteLoggedStart`. Both should call `cycleservice.EngineProfile()` / `HistoriesFromRows`.
- **Test plan:**
  - Adapter unit tests for each input.
  - Integration test: a fixture user with every input; `Load` returns all of them.
  - Assert the query count, so the loader can't grow an N+1.
  - Regression: today's golden sweeps (`TestCycleViewGoldenSweep`, `TestLegacyGoldenSweep`) still pass before the resolver starts reading the new fields.

### IC-04 — Declared onboarding anchor replaces the seed row
```
type: backend   depends_on: [IC-01, IC-02, IC-03, core resolver anchor task]   parallel_group: M9-IC-C
touches: [backend-go/internal/profile/service.go, backend-go/internal/profile/onboarding.go, backend-go/db/queries/profile, backend-go/db/migrations, backend/database/migrations, backend-go/api/openapi.yaml, docs/go-migration/deviations.md]
verify: cd backend-go && make schema-diff && go test ./internal/profile/... && make test-int PKG=./internal/profile/... && make contract ROUTES=profile
model: opus
```
- `profile.Save` writes `onboarding_last_period_start`. `last_period_start` keeps meaning the latest actual start.
- Delete `syncOnboardingPeriodLog` (`profile/service.go:463-513`) and the `Insert/UpdateOnboardingCycleHistory` queries.
- Data migration in goose with a Laravel mirror (rules in §4):
  - An untouched seed is deleted and its start moves into the column.
  - A seed the user edited becomes a confirmed `user_logged` row.
  - `down` restores the rows.
- The resolver's anchor comes from the column (coordinate with report 01: §2.41 `predicted_reference`, `onboarding_based`).
- Add the deviation row and the contract allow-list entries (`profile.yaml`, `cycle.yaml`, `home`).
- **Test plan:**
  - Migration integration test with 3 personas: an untouched seed, an edited seed, and a seed followed by later logs.
  - Profile integration test: a declared LMP creates no `cycle_histories` row.
  - Resolver tests for `onboarding_declared` and the stale anchor.
  - Re-record the contract goldens behind the deviation.

### IC-05 — Bleeding-log → period bridge v6
```
type: backend   depends_on: [IC-01, IC-02]   parallel_group: M9-IC-C
touches: [backend-go/internal/healthlog/cyclehistory.go, backend-go/internal/healthlog/service.go, backend-go/internal/healthlog/entries.go, backend-go/db/queries/healthlog/side_effects.sql, backend-go/internal/cycle/periods, backend-go/internal/cycle/model/logged_start.go, docs/go-migration/deviations.md]
verify: cd backend-go && go test ./internal/healthlog/... ./internal/cycle/... && make test-int PKG=./internal/healthlog/...
model: opus
```
- Rows created by the bridge get `source=bleeding_log` and `is_confirmed=1`, so the engine treats them as actual. Remove the `PromoteLoggedStart` special case, or keep it only for legacy rows (B2).
- Mode guard: in the effective mode `pregnancy`, a `flow` log only raises the alert; there's no period row and the LMP doesn't move (B3). Postpartum follows the IC-01 decision (B8).
- Deleting a day, deleting a log, or clearing `flow` rebuilds the affected `bleeding_log` period from the remaining bleeding days: shrink its end, or delete it when no bleeding day is left. Never touch `user_logged` or button rows (B4, B7).
- Every bridge write calls the shared `periods.recomputeCycleLengths` and `reanchor`, exported once (B5).
- `periods.Update` marks an edited row confirmed `user_logged` (S5).
- Revise D-55 and update `TestStoreSideEffectsMatchLaravel` (`applyD55`).
- **Test plan:** table-driven integration tests for:
  - a forward start
  - extending an open or closed period
  - a back-dated row between two periods, with the next row's `cycle_length` recomputed
  - deleting the only bleeding day
  - clearing the middle day of a 5-day period
  - pregnancy-mode flow creating no row
  - spotting creating no row
  - an edit through `PUT /cycle/period/{id}` confirming the row

### IC-06 — Skipped cycle ("I had no period") endpoint
```
type: backend   depends_on: [IC-02, IC-03]   parallel_group: M9-IC-C
touches: [backend-go/internal/cycle/skips, backend-go/internal/http/routes_cycle.go, backend-go/api/openapi.yaml, backend-go/db/queries/cycle/skips.sql]
verify: cd backend-go && go vet ./... && go test ./internal/cycle/... ./internal/http/... && make test-int PKG=./internal/cycle/skips/...
skills: [new-endpoint]
```
- Add `POST /api/v1/cycle/skips {expected_start}` and `DELETE /api/v1/cycle/skips/{date}`.
- POST is accepted only while the engine reports `period_expected` or overdue for that predicted start. It rejects with 422, never 401.
- Calls `MarkRecalculated`. Documented in OpenAPI.
- The engine rule (excluded from medians and regularity, next start = previous prediction + L, the `skipped_cycle_recorded` warning) belongs to report 01; this task provides the input.
- **Test plan:** handler integration tests (accept, duplicate, not-expected → 422, delete); the OpenAPI route-coverage test.

### IC-07 — One engine cache key for every v6 input
```
type: backend   depends_on: [IC-03]   parallel_group: M9-IC-C
touches: [backend-go/internal/cycle/service/service.go, backend-go/internal/cycle/cache, backend-go/internal/clinicalconfig]
verify: cd backend-go && go test ./internal/cycle/... ./internal/clinicalconfig/... && make test-int PKG=./internal/cycle/cache/...
```
- Replace the ad-hoc key flags (`service.go:191-215`) with an `InputsDigest` over the whole `model.Inputs`, plus `clinicalconfig.Fingerprint` for the `engine.*` keys. Cache schema 3 → 4.
- The engine needs no new `MarkRecalculated` calls. Keep `calculation_version` only as the client-visible signal.
- **Test plan:**
  - For every input in §7, a unit test asserting that changing that input changes the key.
  - Unrelated rows, such as another day's symptoms, leave the key unchanged.
  - Integration test: publishing a clinical-config change makes the next read miss the cache.

### IC-08 — User-local day (device timezone, §2.3)
```
type: fullstack   depends_on: [IC-01, IC-02]   parallel_group: M9-IC-C
touches: [backend-go/internal/platform/civildate, backend-go/cmd/api, backend-go/internal/cycle/service/handlers.go, backend-go/internal/home, backend-go/internal/healthlog, backend-go/internal/fertility, backend-go/internal/analysis, frontend/src/shared/api]
verify: cd backend-go && go test ./internal/platform/... ./internal/cycle/... && cd ../frontend && npm run typecheck && npm run test
```
- The frontend client sends `X-Timezone: <IANA>` (from `Intl.DateTimeFormat().resolvedOptions().timeZone`). The display layer keeps formatting in the user's calendar.
- Middleware validates the zone with `time.LoadLocation` (tzdata is embedded: `cmd/api/main.go:18`), falls back to `Asia/Tehran`, and stores the zone on the profile when it changes.
- Add a `civildate.TodayFor(c)` helper.
- Switch only the engine path and its direct consumers in M9: cycle, home, healthlog date defaults, fertility, analysis, companion, teen, checkups, messages daily. Document that the other ~100 call sites stay on Tehran.
- DB timestamps stay Tehran wall-clock (`backend-go/CLAUDE.md:84-85`).
- **Test plan:**
  - Unit tests at 23:30 Tehran vs 21:00 Europe/Berlin (different days), and with no header (identical to today's behaviour).
  - Contract runs without the header stay byte-identical.
  - Frontend test: the header is sent.

### IC-09a — Calendar month and /home read the v6 engine output
```
type: backend   depends_on: [IC-03, IC-07, core v6 resolver]   parallel_group: M9-IC-D
touches: [backend-go/internal/cycle/service/service.go, backend-go/internal/cycle/view, backend-go/internal/clinicalconfig/registry.go, backend-go/internal/home/sections.go, backend-go/internal/home/context.go, backend-go/internal/home/cycle_overview.go, docs/go-migration/deviations.md]
verify: cd backend-go && go test ./internal/cycle/... ./internal/home/... && make contract ROUTES=cycle,home
model: opus
```
- `/cycle/month` builds days from the v6 resolver, through a range resolve, instead of `legacy.CalculateForDate`.
- `/home` `week_calendar`, `next_period` (with `days_late`, `days_to_period`) and `cycle_prediction` (O−5..O, ovulation with its source) come from the snapshot.
- Remove `view.Predict`'s O+1 window, and `engine.fertile_window.after=1`, or migrate the config.
- Honour the §2.42 suppression: no fertile window under hormonal contraception; a stopped state in pregnancy.
- The health-check "normal" ranges come from §2.27 `clinical_flags` and `cycle_regularity`.
- `HasCycleData` becomes "the engine has an anchor", no longer "the LMP is set".
- **Test plan:**
  - New cross-consumer parity test: 8 personas × 120 days. For each day, `/cycle/month`, `/home` and `cycle_view` must agree on phase, fertile window, next start and lateness.
  - Re-record the goldens behind the deviation.

### IC-09b — Fertility, analysis and health record use the engine's ovulation and regularity
```
type: backend   depends_on: [IC-03, core v6 resolver (BBT/LH sources, §2.27)]   parallel_group: M9-IC-D
touches: [backend-go/internal/fertility, backend-go/internal/analysis, backend-go/internal/healthrecord]
verify: cd backend-go && go test ./internal/fertility/... ./internal/analysis/... ./internal/healthrecord/...
model: opus
```
- `fertility/insights.go:308-334` and `analysis/ttc.go` read the engine's per-cycle ovulation source (BBT > LH > calendar, with `lh_bbt_conflict`) instead of their own copies.
- The BBT rule exists in one place (the engine or a shared package): strict third reading and ≥10 readings.
- `analysis/cycle.go` regularity becomes the engine's §2.27 output, using `insufficient_data`. Agree the enum rename with the frontend task.
- `healthrecord` `last_period_start` = the latest actual start, excluding declared anchors.
- `fertility` stops calling resolve once per day in loops; it uses one range resolve.
- **Test plan:**
  - Analysis and TTC goldens re-recorded with a deviation.
  - Fertility BBT fixtures: a shift whose third reading is below +0.2 (rejected), and 9 readings (unconfirmed).

### IC-09c — Small consumers converge on the snapshot (plus the group-B items in §5)
```
type: backend   depends_on: [IC-03, core v6 resolver]   parallel_group: M9-IC-D
touches: [backend-go/internal/companion/shared, backend-go/internal/teen, backend-go/internal/checkups, backend-go/internal/profile/cyclesettings.go, backend-go/internal/healthlog/cyclehistory.go, backend-go/internal/todo, backend-go/internal/challenges, backend-go/internal/shop/checklist, backend-go/internal/http/routes_shop_checklist.go, backend-go/internal/labs, backend-go/internal/conditions, backend-go/internal/messages/conditionnudges, backend-go/internal/search]
verify: cd backend-go && go test ./internal/companion/... ./internal/teen/... ./internal/checkups/... ./internal/profile/...
model: sonnet (mechanical once IC-03 exists)
```
- Companion `MaxLateDays` reads the engine cap.
- Teen `WeekBucket` reads the v6 `predicted_next_period_start`.
- Checkups use the snapshot profile.
- Cycle settings use the engine's PMS window and metrics.
- `IsLikelyLutealSpotting` reads the engine phase.
- Todo and shop gate on `enums.ResolveLifeMode`, or on the engine's gates, instead of the raw stored mode.
- `challenges.phase` uses `cycleservice.Load` instead of its own copy.
- Shop resyncs its supply reminder on cycle writes.
- Labs ask the engine for the phase on the lab's `taken_on` date; delete `labs.phaseOn`.
- Condition nudges, PMDD/PBAC and search take their cycle bounds from the engine's actual history and its predicted next start; delete their own 45-day, 60-day and profile-length rules.
- **Test plan:** the existing package tests, plus one fixture per package where today's code and the engine differ (manual lengths, a newest bleeding-log period, a 15-day lateness, a pregnant user with no stored `life_mode`). Shop: a period edit moves the stored reminder.

### IC-09d — Messages read the v6 engine output (daily, selected, admin preview)
```
type: backend   depends_on: [IC-03, IC-07, core v6 resolver, IC-01 (content decision)]   parallel_group: M9-IC-D
touches: [backend-go/internal/messages/source.go, backend-go/internal/messages/manager, backend-go/internal/messages/v2.go, backend-go/internal/messages/rules, backend-go/internal/admin/messages/rules.go, backend-go/db/queries/messages/messages.sql, docs/go-migration/deviations.md]
verify: cd backend-go && go test ./internal/messages/... ./internal/admin/messages/... && make contract ROUTES=messages
model: opus (health copy reaches users)
```
- `StoreSource.Cycle` (`source.go:135-160`) is replaced by the snapshot's v6 output. Delete `engineProfile` and `historyFromRow`.
- Map `main_phase` and `subphase` to the existing content keys with `enums.MainPhase.LegacyPhase()`. This needs a **product decision**: v6's fertile phase lasts 6 days while today's legacy "ovulation" content shows for 1 to 2 days.
- Add the v2 facts `main_phase`, `days_late`, `data_quality` and `suggest_pregnancy_test`. Fix `days_to_period` and the O+1 fertile fact. Strip fertility facts when the engine suppresses fertility (teen, menopause, contraception).
- Fill the `pms_start_day` / `top_symptoms` template variables or remove them (`messages/rules/validate.go:13`). Today they are allowed but never produced, so any rule using them is skipped.
- The message *engine* rules belong to the message-engine report; this task only swaps the cycle source.
- **Test plan:**
  - Messages goldens re-recorded behind a deviation.
  - Fixtures: a late user (8 and 15 days), a teen with goal TTC, a pill user, and O+1.
  - The daily message and the home hero must agree on phase for 8 personas × 60 days.

### IC-13 — Fix the `pregnancy_week` fact (independent bug, small)
```
type: backend   depends_on: []   parallel_group: M9-IC-A
touches: [backend-go/internal/messages/v2.go]
verify: cd backend-go && go test ./internal/messages/...
model: sonnet
```
- `v2.go:73` computes `today.DiffDays(lmp)`, which is negative for a past LMP, so `pregnancy_week` is wrong for every pregnant user in `/messages/selected` and the admin preview. Use `lmp.DiffDays(today)/7 + 1`, or better, read the pregnancy calculator's gestational week (`bgi/pregnancy/calc`) so the two never disagree.
- **Test plan:** a unit test with an LMP 10 weeks ago expects week 11. Add a rule-matching test on `pregnancy_week`.

**Cross-report items, not proposed here:** the engine-health metrics replay and its distributions (`bgi/metrics/metrics.go:297-360`), and the engine block on the admin user screens (SPEC §5.2). Both belong to the KPI/admin report. They will need a batch variant of the snapshot (`Outputs(ctx, userIDs, today)`); IC-03 should keep the loader batch-friendly, using `IN (…)` queries and pure compute.

### IC-10 — `user_profiles.user_id` UNIQUE (schema hygiene)
```
type: backend   depends_on: []   parallel_group: M9-IC-A
touches: [backend-go/db/migrations, backend/database/migrations, backend-go/db/queries/profile/profile.sql]
verify: cd backend-go && make schema-diff && make test-int PKG=./internal/profile/...
```
- Pre-check query: count users with more than one row on stage and beta, read-only. Report it before writing the migration.
- Migration: when duplicates exist, fail loudly (preferred), or keep the lowest id after merging its non-null fields (decision). Then add `UNIQUE(user_id)`.
- `INSERT` becomes `INSERT … ON DUPLICATE KEY UPDATE`, and `ORDER BY id LIMIT 1` can stay.
- **Test plan:** migration integration test with a duplicate fixture (expect failure) and a clean fixture; a concurrency test with two parallel first saves leaves one row.

Frontend tasks: see `02b-frontend-consumers.md` §9 and `02c-frontend-impact.md` §F. The final task ids are in `../GAP-ANALYSIS.md`.

