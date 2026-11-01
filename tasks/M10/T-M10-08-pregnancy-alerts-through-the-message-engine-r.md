---
id: T-M10-08
title: Pregnancy alerts through the message engine; retire the v1 alert engine
milestone: M10
type: backend
status: done
depends_on: [T-M10-07, T-M9-05]
parallel_group: M10-B
touches: [backend-go/internal/messages/pregnancyalerts, backend-go/internal/messages/facts/pregnancy_alerts.go, backend-go/internal/pregnancy/alerts, backend-go/internal/pregnancy/handlers.go, backend-go/internal/pregnancy/daylog_bridge.go, backend-go/internal/pregnancy/tools, backend-go/internal/vitals/alerts.go, backend-go/internal/safety, backend-go/db/migrations, backend-go/api/openapi.yaml, docs/engine-v6/clinical-signoff.md, backend-go/internal/http/routes_pregnancy.go, backend-go/internal/pregnancy/repo.go, backend-go/internal/pregnancy/v2/daylog/daylog_int_test.go, backend-go/internal/messages/feed_int_test.go, backend-go/internal/admin/messages/testdata/facts_registry.json, backend-go/contract/allowlist/pregnancy.yaml, backend-go/contract/allowlist/pregnancy-tools.yaml]
skills: []
verify: cd backend-go && go vet ./... && go test ./internal/messages/... ./internal/pregnancy/... ./internal/safety/... ./internal/vitals/... && make test-int PKG="./internal/messages/... ./internal/pregnancy/... ./internal/vitals/... ./db/migrations/..." && make contract ROUTES=pregnancy,pregnancy-v2,pregnancy-tools,vitals
---

# T-M10-08 — Pregnancy alerts through the message engine; retire the v1 alert engine
**Model:** opus (clinical safety). The main session reviews every seeded rule, level and text.

## Why
Two pregnancy alert engines run side by side: v1 `internal/pregnancy/alerts` (hard-coded, called from
`pregnancy/handlers.go:396,501,604` and `daylog_bridge.go:65,81`) and v2 `messages/pregnancyalerts` (DB-configured
detectors, its own `pregnancy_alerts` table, one row per reading). Every v6 change has to land twice, and SPEC §5.4.1
says pregnancy alerts must be ordinary messages in `alerts_inbox` (gap 03 §2.6, gap 04 "parallel/legacy debt"). T-M9-05
made the emergency paths fire and moved their numbers to the registry; this task moves the alerts onto the engine
without losing any of that. DECISIONS.md M-10, H-02; D-165.

## Scope
1. Detectors become fact and evidence producers (`facts/pregnancy_alerts.go`, registered like T-M10-03's files):
   `preg.vomiting_days_7`, `preg.vomiting_severe_7`, `preg.cant_keep_fluids`, `preg.days_since_weight`,
   `preg.week_started_today`, `preg.next_visit_in_days`, `preg.fetal_movement` (none | reduced | low_session, T-M9-05
   rule), `preg.bp_last_category` (normal | high | severe; readings from pregnancy logs and from `/vitals` in
   pregnancy), `preg.bp_high_level` (follow_up | urgent from `pregnancy.bp.high.urgent`, H-02), `preg.gdm_last`
   (in_target | above_target per context), `preg.contractions_511`, `preg.critical_symptom` (bleeding, fluid leakage
   and the other v2 critical items). Thresholds only from the registry keys of T-M9-05. Each producer also returns
   evidence `{what, basis, key}` built from the user's own data («این الگو از ثبت‌های خودته، نه تشخیص»).
2. Data migration seeding one rule per alert (SPEC §5.4.1 table): vomiting (clinical/follow_up; `cant_keep_fluids` →
   safety/urgent), weight missing (product/suggestion), new week (content/info), next visit (content/info), reduced
   fetal movement (safety/urgent), BP high (two rules keyed on `preg.bp_high_level` so the config picks the level),
   BP severe (safety/urgent), GDM (clinical/follow_up), 5-1-1 (safety/urgent), critical symptom (safety/urgent).
   `rule_key = preg.<name>`, copy taken from today's `message_contents` rows, safety rules `locked = 1`. Append each
   seeded rule (key, condition, level, fa text) to `docs/engine-v6/clinical-signoff.md` for the advisor.
3. Emergency guarantee (SPEC §4.2): if a locked `preg.*` safety rule row is missing or inactive, the engine uses the
   code-held definition with embedded fa/en copy and still raises the alert (keeps T-M9-05's guarantee).
4. Data migration: `pregnancy_alerts` rows → `message_inbox` (state mapping: acked → acknowledged, visit note →
   added_to_visit_note, older than 30 days → archived; `legacy_alert_id` kept); idempotent; `down` removes migrated
   rows only.
5. `/pregnancy/v2/alerts*` become adapters over the inbox with the same JSON shape.
6. Retire v1: the `pregnancy/handlers.go` and `daylog_bridge.go` call sites, the kick/contraction tool writes and the
   pregnancy branch of `vitals/alerts.go` call `inbox.EvaluateNow` and return the alerts created by that write in the
   existing `alerts` array shape; delete `internal/pregnancy/alerts` rules and the v2 detector table reads. One alert
   history per user.
7. Safety scenarios (`internal/safety`) drive the new path (reduced kicks, severe BP, BP through `/vitals`, 5-1-1,
   GDM 2 h).

## Tests
- Parity first: a table over the existing fixtures where v1/v2 `Detect()` hits equal the new rule matches — written
  and green **before** v1 is deleted; keep the table as the regression test afterwards.
- Existing `pregnancyalerts` integration tests stay green through the adapter.
- Fetal movement: the safety alert is in `urgent_modal` and in the inbox; BP 165/112 through `/vitals` in pregnancy →
  exactly one urgent inbox row; flipping `pregnancy.bp.high.urgent` changes the 145/92 level.
- A deleted or inactive locked rule still fires with embedded copy (fa and en).
- Migration up/down over fixture rows (each state mapped).
- Contract: `/pregnancy/*-logs` alerts array shape unchanged.

## Out of scope
- Menopause and condition alerts (T-M10-09), the admin screen for pregnancy alert rules (T-M10-13), the alerts UI
  (T-M10-15), lowering any escalation.

## Acceptance
- Verify passes; no code path reads `pregnancy_alerts` for new alerts; PROGRESS.md cites D-165 and lists the seeded
  rules added to clinical-signoff.md.
