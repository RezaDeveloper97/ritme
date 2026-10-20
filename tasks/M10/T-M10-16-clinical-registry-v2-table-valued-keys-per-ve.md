---
id: T-M10-16
title: Clinical registry v2: table-valued keys, per-version source, coverage keys
milestone: M10
type: fullstack
status: todo
depends_on: [T-M10-01, T-M9-05, T-M9-06, T-M9-15]
parallel_group: M10-C
touches: [backend-go/internal/clinicalconfig, backend-go/internal/admin/clinicalcfg, admin-web/src/screens/clinical-config, admin-web/messages/fa.json, admin-web/messages/en.json]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/clinicalconfig/... ./internal/admin/clinicalcfg/... && make test-int PKG=./internal/admin/clinicalcfg/... && cd ../admin-web && npm run typecheck && npm run lint && npm run test
---

# T-M10-16 — Clinical registry v2: table-valued keys, per-version source, coverage keys
**Model:** opus (clinical config). The main session checks every default against SPEC §3 and the cited source.

## Why
SPEC §5.8: every clinical threshold lives in the registry `{key, value, unit, source, version, effective_from,
approvals[]}` and the engines read the version valid on the target date. The registry has about 30 keys (20 before
T-M9-05/06/15); most §3.1–§3.5 numbers are still hard-coded (gap 04 §3 inventory), and tables — IOM bands, the ACOG
re-dating table, screening intervals, the vaccine schedule — cannot be stored because a value is "named numbers"
only (`registry.go:24-40`). `Version` does not carry its `source`, so payloads cannot cite it (`vitals.SourceBP` is a
constant). DECISIONS.md H-07, H-08, H-09, H-12, H-17. No schema change: `clinical_config_versions.value` is JSON text.

## Scope
1. `Def` gains `Kind` (`numbers` | `rows`); a rows key has a row schema (`[]Column{Name, Type: number|string|enum,
   Enum, Min, Max, Nullable}`), `MinRows/MaxRows` and an optional `SortBy` + contiguity rule. Stored value
   `{"rows":[...]}`. Validation on proposal: types, bounds, enum values, sort order, no gaps/overlaps where declared.
2. `Version` carries `Source` (v1 = `Def.Source`, v2+ = the stored `source` column). New getters: `Rows(key, d)`,
   `SourceOf(key, d)`; `Fingerprint` covers rows keys.
3. Keys (version 1 = code default; today's behaviour unless the row says "spec"):
   - `pregnancy.edd {naegele_days 280}`, `pregnancy.edd.cycle_correction {enabled 0}`;
   - `pregnancy.trimester {t2_from_days 98, t3_from_days 196}` (spec; consumers switch in T-M10-18);
   - `pregnancy.redating` rows `{ga_until_days, max_diff_days}` = ACOG CO 700: (62, 5), (97, 7), (111, 7), (153, 10),
     (195, 14), (open, 21) — check the table and its footnote (which GA the band uses) against SPEC §6.2 and write the
     answer in the key's source text;
   - `pregnancy.iom` rows `{bmi_from, bmi_to, total_lo, total_hi, weekly_lo, weekly_hi}` and
     `pregnancy.iom.first_trimester {lo 0.5, hi 2}` (values moved from `analysis/pregnancy_thresholds.go:19-33`);
   - `pregnancy.spotting {until_week 12}`; `epds.schedule {full_days 14, short_days 7}`;
   - `lab.hb {nonpregnant_low 12, pregnant_low 11, pregnant_t2_low 10.5}`, `lab.ferritin.inflamed {low 70}` (the
     existing `lab.ferritin.low` stays), `lab.vitd {deficient_below 12, sufficient_from 20}`,
     `lab.tsh.pregnancy {upper 4.0}`, `lab.hba1c {prediabetes_from 5.7, diabetes_from 6.5}`,
     `lab.ocr {min_confidence 0.9}` (H-08, ships), `lab.borderline {enabled 1, margin 0.10}` (H-07 safe default);
   - `bmi.who {underweight_below 18.5, overweight_from 25, obese_from 30}`;
   - `glucose.contexts {spec 0}` (H-09 safe default), `glucose.diabetic {fasting_lo 80, fasting_hi 130,
     post_meal_max 180}`;
   - `child.growth {low_pct 3, high_pct 97, cross_lines 2}`;
   - `screening.uspstf` rows `{type_key, age_min, age_max, interval_months, last_test_kind, min_bmi}` = the SPEC §3.4
     table (cervical 21–29 / 36 cytology, 30–65 / 36 cytology or 60 HPV/co-test; mammography 40–74 / 24; BP
     screening 18–39 / 36, ≥ 40 / 12; glucose screening 35–70 with BMI ≥ 25 / 36; dental / 6). Nothing reads it until
     T-M10-21 switches the checkups engine from the catalog columns;
   - `child.vaccine_schedule` rows `{vaccine_code, dose, age_months}` = today's `child_vaccines` catalog ages (read the
     seed in `db/migrations/00031_children.sql`), H-17.
   A test pins every v1 value to the constant or seed it replaces, or to the SPEC table for keys nothing read before.
4. Admin API (`internal/admin/clinicalcfg`): list returns kind, schema and source per version; propose accepts rows;
   the two-advisor flow is unchanged.
5. Admin-web clinical-config screen: a table editor for rows keys (add/remove/reorder rows, per-column validation,
   diff against the version in force), the source of each version shown.

## Tests
- Validation per new key: bounds, enum, order, gaps; a rows proposal with an overlapping band → 422.
- `At`/`Rows`/`SourceOf` the day before, on and after `effective_from`.
- v1 defaults equal the old constants and seeds (table test naming each file:line it replaces); fingerprints of the
  numbers keys unchanged by this refactor.
- Admin-web vitest: rows editor round trip and validation messages.

## Out of scope
- Switching consumers to the new keys (T-M10-18 … T-M10-22); approving any v2.

## Acceptance
- Verify passes; every new key is listed in `docs/engine-v6/clinical-signoff.md` with its v1 value and source.
