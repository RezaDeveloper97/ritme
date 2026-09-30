---
id: CB-MENO-01
title: Menopause schema, taxonomy items and seeds
epic: MENO
type: backend
status: todo
depends_on: [CB-CORE-03, B-N3-01, B-N2-03]
parallel_group: MENO-A
touches: [backend-go/db/migrations, backend/database/migrations, backend-go/db/queries/menopause, backend-go/sqlc.yaml, backend-go/seeds, docs/canvas-build/menopause.md]
skills: [new-endpoint]
boards: [nbl_Meno_Stage.dc.html, nbl_Meno_Log.dc.html, nbl_Meno_HotFlash.dc.html, nbl_Meno_Score.dc.html, nbl_Meno_Treatment.dc.html, nbl_Meno_Checkups.dc.html, nbl_Meno_Alert.dc.html, Main.dc.html]
verify: cd backend-go && make sqlc && go vet ./... && make schema-diff
---

# CB-MENO-01 — Menopause schema, taxonomy items and seeds

## Why
bloom gives menopause only a minimal home (B-N2-03). This epic builds the full mode. Built ON TOP of the bloom/ queue: never re-implement what a `B-N*` dependency delivers — extend it.

## Boards
Snapshot in `docs/design/canvas-v1/boards/` (text in `docs/design/canvas-v1/text/`):
- `nbl_Meno_Stage.dc.html`
- `nbl_Meno_Log.dc.html`
- `nbl_Meno_HotFlash.dc.html`
- `nbl_Meno_Score.dc.html`
- `nbl_Meno_Treatment.dc.html`
- `nbl_Meno_Checkups.dc.html`
- `nbl_Meno_Alert.dc.html`
- `Main.dc.html`

## Scope
1. Daily symptoms (13, grouped), bleeding none/spot/bleed and triggers are **log-taxonomy v2 items scoped to menopause** (B-N3-01) — no parallel day-log table.
2. New tables: menopause_profiles (stage peri/meno/post/unsure, last period month, surgical, on_hrt), hot_flashes (started_at, duration_s, severity 1–4, night, sweat, triggers[]), menopause_scores (month, 11 answers 0–4, total/44, domain subtotals 16/16/12), treatment_items (hrt/supplement/lifestyle, schedule, dose text, start, review date, weekly goal), treatment_intakes, side_effect_logs.
3. Catalog seeds: meno_score_items, meno_alerts, meno_tips; M4 checkup catalog rows with audience=menopause for the Checkups board — all copy `[needs clinical review]`.
4. docs/canvas-build/menopause.md: model + stage rule (12 months without period ⇒ menopause).

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- schema-diff green; seeds in both migrations.
- `verify` green.
