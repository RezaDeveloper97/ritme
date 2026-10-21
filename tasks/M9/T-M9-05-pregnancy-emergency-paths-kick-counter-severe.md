---
id: T-M9-05
title: Pregnancy emergency paths: kick counter, severe BP, vitals in pregnancy, GDM
milestone: M9
type: fullstack
status: done
depends_on: []
parallel_group: M9-A
touches: [backend-go/internal/clinicalconfig, backend-go/internal/pregnancy/tools, backend-go/internal/pregnancy/alerts, backend-go/internal/pregnancy/labor, backend-go/internal/messages/pregnancyalerts, backend-go/internal/vitals, backend-go/internal/analysis/pregnancy_thresholds.go, backend-go/internal/safety, frontend/src/screens/log-kick, frontend/src/screens/vitals-add, frontend/messages/fa/pregnancy-tools.json, frontend/messages/en/pregnancy-tools.json, frontend/messages/fa/vitals.json, frontend/messages/en/vitals.json]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/clinicalconfig/... ./internal/pregnancy/... ./internal/messages/pregnancyalerts/... ./internal/vitals/... ./internal/analysis/... ./internal/safety/... && make test-int PKG="./internal/pregnancy/... ./internal/vitals/... ./internal/messages/pregnancyalerts/..." && cd ../frontend && npm run typecheck && npm run test
---

# T-M9-05 — Pregnancy emergency paths: kick counter, severe BP, vitals in pregnancy, GDM
**Model:** opus (clinical safety). The main session reviews every threshold and every alert text.

## Why
Gap report 04 §2 S1 — a patient can miss an urgent prompt:
- A kick session ending with < 10 kicks in 2 h writes status `felt` and raises **nothing**
  (`pregnancy/tools/service.go:322-326`); only manual `reduced/none` fires, and only from week 24
  (`messages/pregnancyalerts/rules.go:287-297`). The nightly safety scenario tests a dashboard counter, not this path.
- Pregnancy BP ≥ 160/110 has no urgent tier in either alert engine (v1 `pregnancy/alerts/alerts.go:293-299`, v2
  `rules.go:249-265`); readings logged in `/vitals` are classified with the non-pregnant table and never reach the
  pregnancy detectors (`vitals/thresholds.go:166-178`).
- GDM alerts use `>` instead of `≥` at 95/140 and have no 2 h (≥ 120) rule (`alerts.go:303`, `rules.go:277`).
- Emergency rules stop firing when their `message_contents` row is missing or disabled (`pregnancyalerts/rules.go:1-10`).
SPEC v6 §3.1, §4.2 ("fixed emergency paths"), §6.1. DECISIONS.md H-02…H-06: additions ship now; reductions stay as
config values with today's behaviour until the advisor signs off.

## Scope
1. Clinical registry keys (`clinicalconfig/registry.go`, group pregnancy, source string per key, lo/hi bounds):
   `pregnancy.bp.high {systolic 140, diastolic 90, urgent 1}` (1 = today's level; the advisor may set 0 = follow_up,
   H-02), `pregnancy.bp.severe {systolic 160, diastolic 110}`, `pregnancy.gdm.targets {fasting 95, one_hour 140,
   two_hour 120}`, `pregnancy.kicks {target 10, window_min 120, from_week 28}`,
   `pregnancy.fetal_movement.manual_from_week {week 24}` (H-04), `pregnancy.contractions_511 {interval_min 5,
   duration_s 45, period_min 60}` (H-06). Defaults equal today's constants except the new escalations.
2. Kick counter: a session that ends — or whose window has expired — with fewer than `target` kicks at or after
   `from_week` raises the urgent `fetal_movement` alert with «همین حالا با پزشک یا ماما تماس بگیر» and 115. Expiry is
   evaluated lazily on the next read/write of kick sessions and in the daily pregnancy-alert evaluation (a session
   started > window ago and never ended counts as ended at `start + window`). The session keeps its real status.
3. Severe BP → urgent (115) in v1 alerts, the v2 `bp_high` rule (two tiers) and the analysis card (registry values).
   ≥ high → level from `pregnancy.bp.high.urgent`.
4. Vitals in pregnancy: when the effective life mode on the reading's date is pregnancy, `vitals` classifies BP with
   the pregnancy table and glucose with the GDM targets (fasting / 1 h / 2 h contexts), and hands the reading to the
   pregnancy alert engine — one alert row, never two. Non-pregnant behaviour unchanged.
5. GDM: fasting ≥ 95, 1 h ≥ 140, new 2 h ≥ 120 → follow_up, in v1, v2 and vitals.
6. Emergency rules always fire: when the copy row of `fetal_movement`, `contractions_511`, `critical_symptom`,
   `bp_high` (severe tier) or `fluid_leakage` is missing or disabled, use embedded fa/en copy (`go:embed`) and still
   raise the alert. Admins can edit copy, not switch the path off.
7. 5-1-1 reads the registry (`pregnancy/labor/labor.go:39-42`), not `message_contents` params.
8. Frontend: after a kick session ends with an urgent alert, the existing urgent alert UI shows immediately (verify;
   wire it if missing); the vitals add screen shows the pregnancy alert for a pregnant user. fa/en, light/dark.
9. Safety scenarios (`internal/safety/scenarios.go`): `reduced_kicks` now drives the real alert path; add
   `pregnancy_bp_severe`, `pregnancy_vitals_bp`, `gdm_two_hour`, `emergency_rule_without_copy`.

## Tests
- Kicks: 28+0 with 9 kicks at 120 min → urgent; 10 at 120 → none; 10 at 121 → urgent; 27+6 with 3 kicks → none; a
  session started 2 h 1 min ago with 4 kicks and never ended → urgent on the next read; manual `reduced` at 26+0 still
  follows `manual_from_week = 24`.
- BP (pregnant): 139/89 none; 140/80 and 120/90 → level from config; 159/109 same; 160/80 and 120/110 → urgent;
  165/112 entered through `/vitals` → one urgent pregnancy alert. Not pregnant: 160/100 → stage2, no pregnancy alert.
- GDM: fasting 94/95, 1 h 139/140, 2 h 119/120 (below → none, at → follow_up).
- Copy row disabled or deleted → the alert still fires with embedded copy in fa and en.
- 5-1-1: 13 contractions every 5 min lasting 60 s over 60 min → met; 6-min spacing → not met; a registry change of
  `duration_s` is honoured.
- Registry: bounds validation per new key; defaults equal the old constants.

## Out of scope
- Moving alerts into the message engine (T-M10-08), postpartum (T-M9-06), lowering any escalation.

## Acceptance
- Every test green; verify passes; PROGRESS.md cites D-134 and lists the advisor-pending keys (H-02, H-04, H-06).
- Ask the user whether to deploy this fix to stage and beta now.
