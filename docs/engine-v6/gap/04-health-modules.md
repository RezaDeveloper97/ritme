# Gap analysis 04 — Health modules vs Engine v6 (§3.1–3.5, §4.2, §5.8, §6.1)

Read-only audit, 2026-10-21, branch `stage`. Paths relative to `backend-go/` unless noted. Spec = `docs/engine-v6/SPEC.md`.

Test run (`go test ./internal/{vitals,pregnancy,postpartum,labs,checkups,children,clinicalconfig,safety}/...`): **all green** (every package `ok`, cached; store packages have no tests). Green means the code matches its *own* tests. Several tests check the pre-v6 behaviour (Laravel parity) and contradict the spec. The trimester and EPDS ≥13 tests are examples.

## 1. Summary table

Status: ✅ compliant · 🟡 partial / boundary or config issue · ❌ missing or contradicts spec

| # | Rule (spec) | Status | Where | Note |
|---|---|---|---|---|
| 1.1 | EDD = LMP+280 (§3.1) | ✅ | `internal/pregnancy/calc/calc.go:173`, `v2/dating.go:74` | Hard-coded 280. |
| 1.2 | Cycle-length EDD correction, default OFF | ✅ (by absence) | — | The option doesn't exist, which matches "off". No setting to turn it on. |
| 1.3 | Ultrasound dating = GA at scan + elapsed days | ✅ | `calc/calc.go:99-104,174-179`, `dating_apply.go:48` | |
| 1.4 | LMP→US replacement table (5/7/7/10/14/21 d) | ❌ | — | Not implemented. The user picks `age_source` and an ultrasound always wins. No discrepancy check. |
| 1.5 | Trimesters ≤13+6 / 14+0–27+6 / ≥28+0 | ❌ boundary bug | `calc/calc.go:131-139` (`weeks <= 12`), `v2/labels.go:107` | 13+0..13+6 is labelled trimester 2. Test `calc_test.go:82` checks `13: 2`. |
| 1.6 | IOM 2009 bands | ✅ values / 🟡 config | `internal/analysis/pregnancy_thresholds.go:19-24`, `analysis/pregnancy.go:81` | Correct values, but hard-coded. Not in the registry. |
| 1.7 | Fetal movement: from 28 w, 10 in ≤2 h, else urgent | ❌ | `pregnancy/tools/service.go:34-35,322-326`; `messages/pregnancyalerts/rules.go:287-297`; `pregnancy/alerts/alerts.go:220-241` | A kick session that ends under 10 kicks in 2 h only writes status `felt`. It raises **no alert**: only a view flag `low_count` (`tools/views.go:89`). Alerts fire only on manual `reduced/none` status, and only **from week 24** (seed params, `alerts.go:222`). |
| 1.8 | Contractions 5-1-1; waters/bleeding → call | 🟡 | `pregnancy/labor/labor.go:39-42,128-157`; `pregnancyalerts/rules.go:157-159` | 5 min / 45 s / 60 min. The params are in admin-editable `message_contents`, not the registry. Waters → `fluid_leakage` emergency (`alerts.go:149`) ✅. |
| 1.9 | Pregnancy BP ≥140/≥90 call doctor; ≥160/≥110 urgent; replaces §3.2 in pregnancy | ❌ | v1 `pregnancy/alerts/alerts.go:293-299`; v2 `pregnancyalerts/rules.go:249-265` + seed `db/migrations/00005_pregnancy_v2.sql:242`; analysis `analysis/pregnancy_thresholds.go:46-50` | No ≥160/110 "urgent" tier in any alert engine; it exists only in the analysis card. The v2 seed marks ≥140/90 as **urgent**, which over-escalates. **Vitals ignore pregnancy:** a pregnant user logging 165/112 in `/vitals` gets `stage2`, no alert (`vitals/thresholds.go:166-178`). Vitals readings never reach the pregnancy detectors, which read `pregnancy_weekly_logs` only. |
| 1.10 | GDM targets fasting <95, 1 h <140, 2 h <120 | 🟡 boundary | analysis `pregnancy_thresholds.go:58-62` ✅ (`<`); alerts `alerts.go:303` & `rules.go:277` use `> 95`, `> 140` | 95 and 140 are out of target but raise no alert. No 2 h (<120) rule in the alert engines. Not pregnancy-aware in vitals. |
| 1.11 | Bleeding: any → discuss; heavy+pain → urgent | 🟡 | `alerts.go:116-131`; v2 `critical_symptom` | Bleeding is "warning"; severe bleeding **or the high-risk profile** gives emergency. "Heavy + pain" isn't modelled as a pair. |
| 1.12 | Postpartum red flags (pad/h, fever ≥38, severe pain, breathlessness → urgent) | ❌ | `postpartum/recovery.go:244-253` | Only `lochia heavy` and `large clots` raise alerts. No fever, pad-per-hour or breathlessness fields. Severe pain is logged but not alerted. |
| 1.13 | Postpartum visits: contact ≤3 w, comprehensive ≤12 w, default reminder 6 w | ❌ | — | No postpartum visit reminders in `postpartum`, `care` or `checkups`. |
| 1.14 | EPDS full 10 every 2 w; ≥10 talk to doctor; ≥13 high | ✅ / 🟡 | `postpartum/epds.go:34-36,121-151`; registry keys `epds.cutoff.*` | Cadence ✅, cut-offs ✅ (config). But `total ≥ 13` also sets `Urgent=true` (`epds.go:147-150`), which opens the 123/115 crisis page. Spec: only Q10 does. A weekly short EPDS-3 between full checks is extra (D-54). |
| 1.15 | EPDS Q10 > 0 → urgent with 123+115 regardless of total | ✅ | `epds.go:144-150`; numbers `postpartum/view.go:17`, `guide/lang/fa/postpartum_copy.json:26` | Tested at total 1 (`epds_test.go:60`). The spec case "Q10=1, total 4" isn't a literal test. |
| 2.1 | BP ACC/AHA categories, higher wins | ✅ | `vitals/thresholds.go:166-178` | Boundaries correct: `>180`/`>120` crisis, `≥140`/`≥90`, `≥130`/`≥80`, `≥120`. Tests `vitals_test.go:23-32` cover 119/79, 120/79, 181/100. **No test for 129/80 or 130/79.** |
| 2.2 | Crisis copy: re-measure 5 min, 115 with symptoms | ✅ / 🟡 | `vitals/lang/fa/vitals.json` `alerts.bp_crisis`, admin row `vitals_alert` | The copy hard-codes «۱۸۰/۱۲۰» and «۵۴». It goes stale if the registry value changes. |
| 2.3 | Resting HR 60–100, rest readings only | ✅ | `thresholds.go:200-216` | Inclusive; contexts resting/after-waking/"" only. |
| 2.4 | Glucose fasting <100 / 100–125 / ≥126 | ✅ | `thresholds.go:56-62,181-197` | Tests 99/100/125/126 at `vitals_test.go:50-53`. |
| 2.5 | Post-meal (non-diabetic) <140 in range, else "discuss"; 2 h OGTT 140/200; random only ≥200+symptoms | 🟡 | `thresholds.go:59-61` | After-meal, bedtime and random all use 140/200 with `high`/`very_high`. A glucometer post-meal ≥200 is labelled like the diagnostic OGTT. Random 150 is `high` (spec: no class). No symptoms input. |
| 2.6 | Diabetic targets 80–130 / <180 (doctor-overridable) | ❌ | — | No diabetes flag and no per-user target override in vitals. |
| 2.7 | Hypo <70 L1, <54 L2 urgent + 15 g rule | ✅ | `thresholds.go:186-189`, alert copy | Registry keys `glucose.low` / `glucose.urgent_low`. |
| 2.8 | HbA1c <5.7 / 5.7–6.4 / ≥6.5 | ❌ in vitals | — | Only as a lab marker, against the sheet or catalog range. No ADA band. |
| 2.9 | mmol/L = mg/dL ÷ 18 | ✅ | `thresholds.go:44` | |
| 2.10 | BMI WHO bands | ✅ / 🟡 | `profile/bmi.go:82-91`, `enums/health_logic.go:25-35` | Duplicated and hard-coded. Pregnancy uses the pre-pregnancy BMI ✅ (`analysis/pregnancy.go:128`). |
| 2.11 | Reports: mean of ≥2 readings on ≥2 days | ❌ | `vitals/reports.go:120-135,248-290` | Averages and classifies from 1 reading. No minimum-data gate. |
| 3.1 | Flow consent→upload→OCR→confirm→rules→LLM | ✅ | `labs/labs.go:7-20`, `handlers.go:310`, `worker.go` | Async worker. |
| 3.2 | Range priority sheet → "general range" → none | ✅ / 🟡 | `labs/status.go:20-24`, `catalog.go:57-62` | `RangeTypical` exists. Check the UI shows «محدوده عمومی». |
| 3.3 | OCR confidence <0.9 amber, excluded until confirmed | 🟡 | `labs/labs.go:89` (`LowConfidence = 0.7`) | Threshold is 0.7, not 0.9. Confirmation is all-or-nothing per report (verify), not per value. |
| 3.4 | Only in/out of range | 🟡 | `labs/status.go:28,41-58` | Extra `borderline_low/high` (±10 %) categories. They aren't in the spec. |
| 3.5 | LLM must not change categories, name diseases or give doses; diagnostic-word output filter | 🟡 | prompt `labs/interpret.go:234-245`; output `cleanSummary` `:361` | Prompt-level guardrails only. Output filter strips URLs and redacts names; there is **no diagnostic-word filter**. Free text, not the spec's JSON {summary, out-of-range explanations, doctor questions}. |
| 3.6 | Hb <12 (non-preg), <11 preg (T2 <10.5); ferritin <15 / <70 with inflammation; vit D <12 deficient, ≥20 sufficient; pregnancy TSH upper 4.0 | ❌ | seed `db/migrations/00034_labs.sql:123-126` (Hb 12–15.5), `:153` (ferritin 15–150), `:163` (TSH 0.4–4.0), `:173` (vit D **30–100**) | No pregnancy- or inflammation-specific fallback. The vitamin D fallback contradicts IOM 2011: 25 ng/mL shows below range, but the spec says sufficient. The Endocrine 2024 note is absent. Registry key `lab.ferritin.low` exists but **no engine reads it** (`clinicalconfig/registry.go:131-133`). |
| 3.7 | "name and number not sent" only if truly anonymised | ✅ (honest) | `internal/ai/pii.go:12-34`; `consent/lang/fa/consent.json` `ai_lab_analysis` | Images and PDFs are sent as-is. The consent text correctly says names are removed **from texts** and asks the user to cover printed details. Don't add the stronger sentence unless image masking is built. |
| 4.1 | USPSTF intervals table | ❌ / 🟡 | seed `db/migrations/00003_checkups.sql:86-120` | Pap 36 mo for ages 21–65, with no HPV 30–65 / 5 y split. Mammography 12–24 mo from 40 with no max age (spec 40–74 / 24 mo). **No BP-screening** or **glucose-screening** (35–70, overweight) types. Dental 6 ✅. `clinical_breast_exam` and `blood_test` yearly are not in the spec. |
| 4.2 | Next due = last + interval | ✅ | `checkups/engine/engine.go:10-13,327-331` | User override via `next_due_on` ✅. |
| 4.3 | "Needs follow-up" disables default interval | ❌ | — | No follow-up status in the engine. |
| 4.4 | Breast awareness replaces monthly self-exam | ❌ | seed `breast_self_exam` monthly, cycle days 7–10 | v6 change not applied. |
| 4.5 | Pregnancy: non-urgent screenings labelled "after pregnancy" | 🟡 | `engine.go:27-28` (`hide_in_pregnancy`) | They are hidden, not labelled. |
| 5.1 | WHO LMS percentiles; <P3 / >P97 → discuss | ✅ | `children/growth/growth.go:23-34,98`; `growthview.go:146-175` | `InBand` is P3 ≤ p ≤ P97 inclusive, so the strict `<` / `>` in the spec holds. |
| 5.2 | Crossing 2 major percentile lines between two measurements | ❌ | — | Only the latest value is checked. |
| 5.3 | CDC 2022 milestones (2,4,6,9,12,15,18,24,30 mo; 3,4,5 y) | 🟡 | seed `db/migrations/00031_children.sql` | Bands are 2,3,4,6,9,12,18,24,36,48,60. **15 and 30 months are missing.** The 3-month band isn't in CDC 2022. |
| 5.4 | Versioned vaccine schedule config | 🟡 | `children/catalog.go:13-24` (`child_vaccines`) | Admin-editable catalog, but no version or effective date. Schedule changes rewrite history. |
| 5.5 | Feeding/sleep/diapers: no classification | ✅ | `internal/babylog` | |
| 6.1 | Fixed emergency paths without AI | 🟡 | vitals `alerts.go`; EPDS; pregnancy alert engine | No AI on these paths ✅. But the pregnancy rules (`fetal_movement`, `contractions_511`, `critical_symptom`, `bp_high`) **don't fire without a live `message_contents` row**, and admins can turn them off (`pregnancyalerts/rules.go:1-10`). No glucose path in pregnancy. No fever path. Kicks under 10 aren't a path at all. |
| 6.2 | Numbers 115/123 | ✅ / 🟡 | vitals copy (115), EPDS urgent (115, 123, 1480) | Pregnancy v1 alert copy mostly says "go to emergency" without the number. |
| 6.3 | `consent(user_id,type,version,granted_at,revoked_at,scope_json)` | 🟡 | `db/migrations/00012_privacy_support.sql:20-32` + `00032_ai_platform.sql:16` | Has user, consent, version, granted_at and revoked_at. **No `scope_json`.** UNIQUE(user, consent) means one row: grant history is overwritten. Types missing from the catalog: terms + health processing, doctor share per booking, companion and family, PDF link (some live in other tables). |
| 6.4 | No paywall on emergency paths | ✅ | `http/routes_vitals.go`, postpartum, pregnancy tools | No Plus gate. Labs is Plus-gated, which is fine (not an emergency path). |
| 7.1 | Registry `{key,value,unit,source,version,effective_from,approvals[]}` | ✅ core / 🟡 coverage | `internal/clinicalconfig/*`, `db/migrations/00063_clinical_config.sql`, `admin/clinicalcfg/clinicalcfg.go` | Two distinct advisors, no self-approval, `effective_from ≥ today`, audited ✅. Only **20 keys**. Most §3.1/3.3/3.4 numbers are outside it (§3 below). |
| 7.2 | Value valid on target_date | ✅ | `clinicalconfig/store.go:137-146`; vitals `Reading` uses its own day | |
| 7.3 | Immutable audit | 🟡 | `admin/httpadmin/audit.go:27-60` (`admin_audit_logs`) | No delete query exists, but nothing enforces immutability in the DB (no trigger, no REVOKE). |
| 7.4 | Nightly automatic test of all emergency scenarios | 🟡 | `internal/safety/scenarios.go:63-109,178-191`; CI `.github/workflows/safety-scenarios-nightly.yml` | 16 scenarios on pure functions. `reduced_kicks` tests a dashboard counter, not an alert path. No scenarios for pregnancy BP, pregnancy bleeding, waters, postpartum fever or live rule-row presence. `epds_high_total_urgent` checks the off-spec behaviour. The result lives in process memory only (lost on restart, per instance). |

## 2. Findings (by severity)

**S1: safety-relevant (patient could miss an urgent prompt)**
1. **Reduced fetal movement from the kick counter is silent.** `writeFetalDay` maps a failed session (<10 in 2 h) to status `felt` (`pregnancy/tools/service.go:322-326`). The v2 `fetal_movement` rule only fires on `reduced/none` (`rules.go:287-297`, seed `from_week:24`). The safety scenario `reduced_kicks` checks `safety.ReducedKicks`, which only feeds the dashboard count (`safety/scenarios.go:21-25`). The nightly test passes while the path doesn't exist.
2. **Severe-range BP in pregnancy (≥160/≥110) has no urgent path.** v1 `alerts.HighBloodPressure` only has ≥140/90 at "warning". v2 `bp_high` has a single tier. Readings logged in the vitals module are classified with the non-pregnant table and never reach the pregnancy engine.
3. **Postpartum red flags missing:** fever ≥38 °C, a pad an hour, severe pain and breathlessness (`recovery.go:244-253`).
4. **Emergency rules can be disabled from admin.** A pregnancy alert rule without a live `message_contents` row "does not fire" (`pregnancyalerts/rules.go:5-7`). Its numbers (`systolic_min`, `from_week`, `interval_max_minutes`…) are editable without the 2-approval workflow, without a source and without target-date versioning. That breaks §4.2 ("fixed") and §5.8.
5. **Trimester boundary bug:** weeks 13+0–13+6 are shown as T2 (`calc.go:134`, `v2/labels.go:107`). That affects content, analysis "symptoms by trimester" and Hb trimester logic once it's added.

**S2: spec-contradicting classification**
6. GDM alert boundaries use `>` instead of `≥` (95, 140). There's no 2 h <120 rule. In vitals, glucose isn't pregnancy-aware.
7. EPDS total ≥13 sets `Urgent` and opens the crisis page. Spec: ≥13 = "high probability, talk to doctor"; only Q10 is urgent. Needs a medical decision (§4).
8. Vitamin D fallback range 30–100 contradicts IOM 2011 (<12 deficient, ≥20 sufficient). Hb fallback ignores pregnancy (<11; T2 <10.5). Ferritin ignores inflammation (<70). There's no pregnancy TSH logic. The lab ranges live in `catalog_items` (admin-editable, unversioned, no approvals).
9. Glucometer post-meal, bedtime and random readings get diagnostic-style 200 cut-offs; spec says post-meal <140 or "discuss", random only ≥200 + symptoms. Diabetic targets and HbA1c bands are missing.
10. Vitals reports classify from one reading; spec says ≥2 readings on ≥2 days (also §4.1 "فشار و قند").
11. OCR low-confidence threshold is 0.7 vs 0.9. There's an extra "borderline" category (±10 %). There's no diagnostic-word output filter on the LLM summary.

**S3: feature gaps**
12. No LMP→ultrasound replacement table. No postpartum visit reminders (3 w / 6 w / 12 w).
13. Checkups catalog not on USPSTF v6: no HPV split, no BP or glucose screening, mammography has no upper age, breast self-exam is not yet breast awareness, no "needs follow-up", pregnancy items hidden instead of labelled.
14. Children: no "crossing 2 major percentile lines" rule. Milestone bands are missing 15 and 30 months. The vaccine schedule isn't versioned.
15. Consent: no `scope_json` and no grant history (single row per type).

**Parallel/legacy debt:** two pregnancy alert engines run side by side. v1 `internal/pregnancy/alerts` is hard-coded, Laravel-parity and called from `pregnancy/handlers.go:396,501,604` and `daylog_bridge.go:65,81`. v2 `messages/pregnancyalerts` is DB-configured. Their thresholds differ (week 24 in both, but different levels). Any v6 change has to land in one place, and the other should be retired.

## 3. Hard-coded clinical threshold inventory

| file:line | value | used for | proposed registry key |
|---|---|---|---|
| `internal/pregnancy/calc/calc.go:173,179,185`; `v2/dating.go:74` | 280 d | Naegele EDD | `pregnancy.edd.naegele_days` (+ `pregnancy.edd.cycle_correction` bool, default 0) |
| `internal/pregnancy/calc/calc.go:134,136` | ≤12, ≤27 weeks (**wrong: 13**) | trimester | `pregnancy.trimester {t2_from_days:98, t3_from_days:196}` |
| — (missing) | 5/7/7/10/14/21 d by GA band | US re-dating | `pregnancy.redating.discrepancy` |
| `internal/pregnancy/calc/calc.go:229,232` | week 18 / 24 | fetal-movement tracking active / required | `pregnancy.fetal_movement.from_week` (28) |
| `internal/pregnancy/alerts/alerts.go:222` | week 24 | v1 fetal alert | same as above |
| `internal/pregnancy/alerts/alerts.go:298` | 140 / 90 (≥) | v1 pregnancy BP | `pregnancy.bp.high` |
| `internal/pregnancy/alerts/alerts.go:303` | >95, >140 | v1 GDM | `pregnancy.gdm.targets {fasting:95, one_hour:140, two_hour:120}` |
| `internal/pregnancy/alerts/alerts.go:133` | week ≤12 | spotting escalation | `pregnancy.spotting.until_week` |
| `internal/analysis/pregnancy_thresholds.go:47-50` | 140/90, 160/110 | analysis pregnancy BP | `pregnancy.bp.high`, `pregnancy.bp.severe` |
| `internal/analysis/pregnancy_thresholds.go:59-61` | 95/140/120 | analysis GDM | `pregnancy.gdm.targets` |
| `internal/analysis/pregnancy_thresholds.go:20-23,28-33` | IOM bands, 0.5–2 kg T1, wk 13/40 | weight gain | `pregnancy.iom.bands` (tabular → needs a list-valued key, see §5) |
| `internal/analysis/pregnancy_thresholds.go:80-82` | 10 kicks / 120 min / wk 28 | analysis kicks | `pregnancy.kicks {target:10, window_min:120, from_week:28}` |
| `internal/pregnancy/tools/service.go:34-35` | 10 / 120 min | kick counter | `pregnancy.kicks` |
| `internal/pregnancy/labor/labor.go:40-42` | 5 min / 45 s / 60 min (defaults; live from message_contents) | 5-1-1 | `pregnancy.contractions.511` |
| `db/migrations/00005_pregnancy_v2.sql:242-247` (+ contractions/critical rows) | params `systolic_min 140`, `diastolic_min 90`, `fasting_max 95`, `post_meal_max 140`, `from_week 24`, `spotting_until_week 12` | v2 alert rules (admin-editable, no approvals) | move to the keys above; `message_contents` keeps copy only |
| `internal/postpartum/recovery.go:244-253` | `heavy`, `large clots` | postpartum alerts | `postpartum.redflags {fever_c:38, pads_per_hour:1}` |
| `internal/postpartum/epds.go:34-36` | 7 / 14 / 14 days | EPDS cadence | `epds.schedule` |
| `internal/postpartum/view.go:17`, copy | 123 / 115 | emergency numbers | `emergency.numbers` (non-clinical config, but versioned) |
| `internal/vitals/thresholds.go:44` | 18.0 | mmol factor | keep as a constant (physics), document |
| `internal/vitals/thresholds.go:56-62` (via registry) | bedtime/random share the post-meal band | context mapping | `glucose.random {very_high_from:200}`; `glucose.post_meal_glucometer {target_max:140}` |
| — (missing) | 80–130 / <180 | diabetic targets | `glucose.diabetic_targets` (user override on top) |
| — (missing) | 5.7 / 6.5 % | HbA1c | `lab.hba1c` |
| `internal/profile/bmi.go:84-88`; `internal/enums/health_logic.go:27-31` | 18.5 / 25 / 30 | BMI | `bmi.who` |
| `internal/labs/labs.go:89` | 0.7 | OCR low confidence | `lab.ocr.min_confidence` (0.9) |
| `internal/labs/status.go:28` | 0.10 | borderline margin | drop, or `lab.borderline_margin` with medical OK |
| `db/migrations/00034_labs.sql:123-126,153,163,173` (catalog `typical`, `critical`) | Hb 12–15.5 (urgent <7), ferritin 15–150, TSH 0.4–4.0, vit D 30–100 | lab fallback ranges | `lab.hb {nonpregnant_low:12, pregnant_low:11, pregnant_t2_low:10.5}`, `lab.ferritin {low:15, inflamed_low:70}` (key exists, unused), `lab.vitd {deficient_below:12, sufficient_from:20}`, `lab.tsh.pregnancy {upper:4.0}` |
| `db/migrations/00003_checkups.sql:86-120` (`checkup_types.interval_months`, `age_min/max`) | Pap 36 / 21–65, mammo 12–24 / 40+, dentist 6… | screening intervals | `screening.<type> {age_min, age_max, interval_months}` (or version `checkup_types` rows with the same approval flow) |
| `internal/children/growth/growth.go:23-34` | P3 / P97 | growth band | `child.growth.band {low_pct:3, high_pct:97, cross_lines:2}` |
| `catalog_items` group `child_vaccines` | schedule | vaccines | versioned `child.vaccine_schedule@vN` |
| `vitals/lang/*/vitals.json` `alerts.*`; `message_contents vitals_alert` | «۱۸۰/۱۲۰», «۵۴» in text | copy | copy should interpolate `{threshold}` from the registry |

Already registry-driven ✅: `bp.elevated/stage1/stage2/crisis`, `glucose.urgent_low/low/fasting/post_meal`, `hr.resting`, `epds.cutoff.likely/possible/short_elevated`, `cycle.figo.*`, `engine.*`, `lab.ferritin.low` (unused).

## 4. Ambiguities needing a user / medical-advisor decision (recommended default)

1. **EPDS ≥13 crisis page.** Should total ≥13 without Q10 open the 123/115 page? *Recommend: no.* Show "high probability, talk to your doctor soon" (follow-up tone); the crisis page is for Q10 > 0 only, as the spec says. Change `safety` scenario `epds_high_total_urgent` to match.
2. **Pregnancy BP ≥140/90 level.** v2 seed says "urgent"; spec says "call doctor". *Recommend:* ≥140/90 → follow_up ("call your doctor today"); ≥160/110 → urgent (115).
3. **Vitals in pregnancy.** Should vitals readings drive the pregnancy BP/GDM rules? *Recommend: yes.* Spec says the pregnancy table replaces §3.2. One classifier picks the table by the life mode on the reading's date.
4. **Fetal movement start week** (24 now vs 28 in spec). *Recommend 28.* Before 28, keep logging without urgency. Ask the advisor whether "none" at 24–27 should stay a warning.
5. **Kick session <10 in 2 h → urgent alert automatically?** *Recommend:* yes, with the «همین حالا با پزشک یا ماما تماس بگیر» copy, fired at session end or when the 2 h window expires.
6. **Borderline lab category (±10 %).** Not in the spec. *Recommend:* remove it, or keep it only as a visual hint under "out of range", never as a class.
7. **OCR threshold 0.9 and per-value confirmation.** *Recommend:* 0.9 from the registry. Values below it are excluded from rules and AI until the user confirms that specific value.
8. **Bedtime glucose context** (not in spec). *Recommend:* treat it like random: no class below 200.
9. **Breast self-exam rows already recorded by users.** *Recommend:* migrate `breast_self_exam` to an informational «آگاهی از سینه» item with no due date. Keep history read-only.
10. **Pap/HPV choice at 30–65.** *Recommend:* one type whose interval depends on the last test kind (cytology 36 mo, HPV/co-test 60 mo).
11. **Tabular thresholds (IOM, re-dating, screening, vaccines) in the registry**, whose values are named numbers today. *Recommend:* allow list-valued keys (JSON rows validated by a per-key schema), with the same 2-approval flow.
12. **Consent history.** *Recommend:* append-only `consent_events` plus a current-state view, and add `scope_json`. Needs the user's OK because it changes `user_consents` (shared with Laravel until cutover; D-entry needed).

## 5. Architecture & performance

- **Config snapshot is good.** `clinicalconfig` keeps an atomic in-memory snapshot, refreshed in the background every 60 s and immediately on approval in-instance (`store.go:29-31,123-133`). Lookups are lock-free and allocation-light. Classification is already a pure function over `Thresholds` (vitals, EPDS). Gaps: no cross-instance invalidation (up to 60 s stale; fine, since `effective_from ≥ today`). `Version` carries no `source`, so payloads can't cite the per-version source; `vitals.SourceBP` is a constant.
- **Pregnancy alert engine is DB-heavy per write.** `Engine.evaluate` loads the `message_contents` group plus about 7 range queries per evaluation (`pregnancyalerts/engine.go:63,140-200`). Then it runs a dedupe count, insert and re-read per hit (`:395-424`, an N+1 per hit). It runs synchronously on every log or kick write. Fine at current scale. Once thresholds move to the registry, cache the copy rows (they're already "rendered when read") and batch the dedupe.
- **Vitals alert copy**: `AlertCopy` is a DB read on the request path only when a reading trips (`vitals/handlers.go:162`). OK.
- **Labs AI is asynchronous** (worker, `labs/worker.go`, 60 s timeout, Plus quota refund on failure). Nothing synchronous on the request path ✅.
- **Two pregnancy alert engines** (v1 hard-coded, v2 data-driven) double the surface for every v6 change. Consolidate into v2, with numbers from the registry and copy from `message_contents`.
- **Safety self-test** runs in-process (`safety.Loop`, started from `http/routes_admin_safety.go:27`). Every API instance runs it, results aren't persisted, and it tests pure helpers, not the live rule set (rule-row presence, copy present, numbers 115/123 present). Move it to an asynq periodic task that persists `safety_scenario_runs` and asserts end-to-end paths.
- **Audit immutability** is convention only. Consider a DB `BEFORE DELETE/UPDATE` trigger on `admin_audit_logs` and `clinical_config_approvals` (MariaDB supports `SIGNAL`).

## 6. Proposed task breakdown (M9, web + backend only; no Android, no prod cutover)

**T-M9-H01 — Registry v2: coverage, list-valued keys, per-version source**
- Add keys from §3: pregnancy BP high/severe, GDM, kicks, contractions 5-1-1, fetal from-week, trimester, Naegele, re-dating table, IOM bands, postpartum red flags, EPDS schedule, lab Hb/ferritin/vitD/TSH-preg/HbA1c, OCR confidence, BMI, glucose random / glucometer post-meal / diabetic targets, growth band.
- Support list-valued keys (schema-validated rows) and expose the `source` of the version in force through `clinicalconfig.At`.
- touches: `internal/clinicalconfig/*`, `internal/admin/clinicalcfg/*`, `db/migrations/00xxx_clinical_config_list.sql` (+ Laravel twin per schema-diff rule), `api/openapi.yaml`, admin-web clinical-config screen.
- deps: none.
- tests: `Validate` order and range for every new key; `At` on the day before, on, and after `effective_from`; v1 equals the old constants (byte-identical fingerprints).

**T-M9-H02 — Trimester + Naegele + ultrasound re-dating (ACOG CO 700)**
- Fix `calc.Trimester` to completed days: ≤13+6 → T1, 14+0–27+6 → T2, ≥28+0 → T3. Fix `v2/labels.go:107`.
- Re-dating: when a scan date and GA are given and an LMP exists, compute the discrepancy and replace the LMP dating only when it exceeds the band threshold. Return `{replaced, discrepancy_days, band}`.
- Cycle-length correction option, default off.
- touches: `internal/pregnancy/calc`, `pregnancy/v2/dating.go`, `dating_apply.go`, profile PUT handler, frontend pregnancy setup copy.
- deps: H01.
- tests: trimester 13+6 → 1, 14+0 → 2, 27+6 → 2, 28+0 → 3. Re-dating at 8+6 with 5 d → keep, 6 d → replace. At 9+0 with 7 → keep, 8 → replace. At 15+6 with 8 → replace. At 16+0 with 10 → keep, 11 → replace. At 27+6 with 14 → keep, 15 → replace. At 28+0 with 21 → keep, 22 → replace. EDD for LMP 2026-01-01 is 2026-10-08.

**T-M9-H03 — One pregnancy-aware vitals/alert classifier (BP + GDM)**
- Vitals classification picks the pregnancy table when the life mode on the reading date is pregnancy. ≥140 or ≥90 → follow_up "call doctor"; ≥160 or ≥110 → urgent (115). GDM fasting ≥95, 1 h ≥140, 2 h ≥120 → follow_up.
- Vitals readings feed the pregnancy alert engine. Retire the v1 `pregnancy/alerts` BP and sugar rules.
- touches: `internal/vitals/{thresholds,alerts,model}.go`, `internal/messages/pregnancyalerts/rules.go`, `internal/pregnancy/alerts`, `internal/analysis/pregnancy_thresholds.go`, migration updating the seed params to copy-only.
- deps: H01; decisions §4.2 and §4.3.
- tests: pregnant 139/89 none; 140/80 and 120/90 follow_up; 159/109 follow_up; 160/80 and 120/110 urgent; non-pregnant 160/100 is stage2 and not urgent. GDM fasting 94 ok, 95 flag; 1 h 139 ok, 140 flag; 2 h 119 ok, 120 flag.

**T-M9-H04 — Fetal movement and contraction emergency paths**
- A kick session that ends, or reaches 2 h, with fewer than 10 kicks from week 28 raises the urgent `fetal_movement` alert.
- Manual `reduced/none` also fires from week 28.
- The 5-1-1 rule reads registry params. Waters and bleeding go to "call the maternity ward".
- Rules can't be disabled: a missing or disabled row falls back to embedded copy and still fires.
- touches: `pregnancy/tools/service.go`, `messages/pregnancyalerts/{rules,engine}.go`, seed migration, `safety/scenarios.go`.
- deps: H01; decisions §4.4 and §4.5.
- tests: week 28+0 with 9 kicks at 120 min → urgent; 10 kicks at 120 min → none; 10 kicks at 121 min → urgent; week 27+6 with 3 kicks → none. 5-1-1: 13 contractions every 5 min, 60 s each, over 60 min → met; 6-min spacing → not met. Disabled row → still fires.

**T-M9-H05 — Postpartum red flags and visit reminders**
- Add recovery fields: temperature (°C), pads per hour, severe pain, breathlessness. Any red flag → urgent (115).
- Visit reminders: contact by day 21, comprehensive by day 84, default reminder at day 42 (registry).
- EPDS: total ≥13 is no longer urgent (pending §4.1).
- touches: `internal/postpartum/{recovery,epds,guide}.go`, taxonomy, care or reminders hook, frontend recovery sheet (fa+en), migration.
- deps: H01; decision §4.1.
- tests: temperature 37.9 none, 38.0 urgent; pads per hour 1 urgent; EPDS Q10=1 with total 4 → urgent (115, 123); total 9 → low; 10 → possible, not urgent; 13 → likely, not urgent; 30 with Q10=0 → not urgent.

**T-M9-H06 — Vitals §3.2 completion**
- Glucometer post-meal: <140 in range, else "discuss" (no 200 class). Random and bedtime: no class below 200.
- Diabetic targets 80–130 and <180 behind a profile diabetes flag, with a per-user doctor override.
- HbA1c bands. Reports gated on ≥2 readings on ≥2 days. Copy interpolates thresholds.
- touches: `internal/vitals/*`, profile flag, `api/openapi.yaml`, frontend vitals hub and report.
- deps: H01.
- tests (§6.1 boundaries): BP 119/79 normal; 120/79 elevated; 130/79 stage1; 129/80 stage1; 139/89 stage1; 140/89 stage2; 180/120 stage2; 181/100 crisis; 150/121 crisis. Glucose fasting 99 in range, 100 high, 125 high, 126 very high; 69 low; 53 urgent; 54 low. Post-meal 139 vs 140. Random 199 unclassified. HbA1c 5.6, 5.7, 6.4, 6.5. Report with 2 readings on 1 day → "not enough data".

**T-M9-H07 — Labs v6 rules**
- OCR threshold 0.9 with per-value confirmation; values below it are excluded from rules and LLM until confirmed.
- Mode-aware fallback ranges from the registry: Hb by pregnancy and trimester, ferritin with inflammation (CRP flag), vitamin D 12/20 plus the Endocrine 2024 note, pregnancy TSH upper 4.0. Label them «محدوده عمومی».
- Remove the borderline class (§4.6).
- Add a diagnostic-word output filter (fa+en list; on a hit, fall back to the rules summary).
- LLM JSON output {summary, out_of_range[], doctor_questions[]}.
- touches: `internal/labs/{status,interpret,views,labs,context}.go`, seed migration, frontend lab verify and result screens.
- deps: H01; H02 (trimester).
- tests: Hb 11.9 non-pregnant → low, 12.0 → in range; pregnant T1 10.9 → low, 11.0 → in range; T2 10.4 → low, 10.5 → in range. Ferritin 14 → low, 15 → in range; with inflammation 69 → low, 70 → in range. Vitamin D 11.9 → deficient, 12–19.9 → low, 20 → sufficient. Confidence 0.89 → excluded, 0.9 → included. A summary containing «دیابت داری» → filtered to the rules summary.

**T-M9-H08 — Checkups USPSTF v6 catalog**
- Pap 21–29 every 3 y. HPV/co-test 30–65 every 5 y (or cytology every 3 y). Mammography 40–74 every 2 y. BP screening 18–39 every 3–5 y, ≥40 yearly. Glucose 35–70 if BMI ≥25, every 3 y. Dental 6–12 months.
- Breast awareness replaces the monthly self-exam.
- "Needs follow-up" finding disables the default interval.
- In pregnancy, label non-urgent items "after pregnancy" instead of hiding them.
- Intervals versioned (registry or versioned rows).
- touches: `internal/checkups/{engine,service,labels}`, migration, admin-web checkup catalog, frontend checkups screen.
- deps: H01; decisions §4.9 and §4.10.
- tests: age 20 → Pap not yet, 21 → due; 29 → Pap (3 y), 30 → HPV (5 y); 65 → included, 66 → excluded. Mammography at 39 / 40 / 74 / 75. Glucose at age 35 with BMI 24.9 vs 25.0. Next due = last + interval (end-of-month clamp). Follow-up → no due date. Pregnant → label `after_pregnancy`.

**T-M9-H09 — Child growth crossing rule, CDC 2022 bands, versioned vaccines**
- Flag a crossing of ≥2 major percentile lines (3/15/50/85/97) between consecutive measurements.
- Add the 15- and 30-month milestone bands. Drop the 3-month band or mark it non-CDC.
- Vaccine schedule versioned with `effective_from`.
- touches: `internal/children/{growth,growthview,milestones,vaccines,catalog}.go`, migration, frontend child growth chart.
- deps: H01.
- tests: P2.9 → discuss, P3.0 → in band, P97.0 → in band, P97.1 → discuss. P60 to P10 (crosses 50 and 15) → discuss; P60 to P20 (crosses only 50) → none.

**T-M9-H10 — Consent model v6**
- Add `scope_json` and append-only consent events with history.
- Add the missing types: terms + health data at signup (mandatory), doctor share per booking, companion/family per member and scope, PDF link (7 days, revocable).
- touches: `internal/consent/*`, `internal/profile` consents, migration (+ Laravel twin), deviations entry, frontend privacy screen.
- deps: decision §4.12.
- tests: grant → revoke → re-grant keeps 3 events; a stale version answers 403 `outdated`; scope mismatch → 403.

**T-M9-H11 — Emergency-path guarantees and nightly self-test v2**
- Scenarios for every §4.2 path, end to end through the real rule set: BP crisis, glucose <54, reduced kicks, 5-1-1, heavy postpartum bleeding / fever, EPDS Q10, pregnancy BP ≥160/110.
- Assert copy present and numbers 115/123 present.
- Persist runs (asynq periodic task + `safety_scenario_runs`) and alert on failure.
- Add a no-paywall assertion test for these routes.
- Add DB triggers making `admin_audit_logs` and `clinical_config_approvals` append-only.
- touches: `internal/safety/*`, `cmd/worker`, migration, admin-web safety dashboard.
- deps: H03, H04, H05, H06.
- tests: each scenario at threshold−1, threshold and threshold+1 against the registry value of the day. A disabled rule row still passes. `UPDATE` or `DELETE` on the audit table → error.

**T-M9-H12 — Retire v1 pregnancy alert engine**
- Route the `pregnancy/handlers.go:396,501,604` and `daylog_bridge.go` callers to the v2 engine.
- Delete the hard-coded rules and keep one alert history.
- touches: `internal/pregnancy/{alerts,handlers,daylog_bridge}.go`, contract cases.
- deps: H03, H04.
- tests: contract diff on `/pregnancy/*-logs` responses (alerts array shape); no duplicate alerts for one log.

Suggested order: H01 → (H02, H06, H09, H10 in parallel) → H03 → H04 → H05 → H07 → H08 → H12 → H11.
