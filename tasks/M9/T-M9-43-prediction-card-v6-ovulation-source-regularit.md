---
id: T-M9-43
title: Prediction card v6: ovulation source, regularity badge, engine ranges
milestone: M9
type: frontend
status: done
depends_on: [T-M9-42]
parallel_group: M9-C
touches: [frontend/src/screens/home, frontend/src/entities/cycle/api/overview.ts, frontend/messages/fa/home.json, frontend/messages/en/home.json, backend-go/resources/translations/fa/home.json, backend-go/resources/translations/en/home.json]
skills: []
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run copy:check -- home
---

# T-M9-43 — Prediction card v6: ovulation source, regularity badge, engine ranges
**Model:** sonnet; the main session reviews health copy.

## Why
The ovulation row always says «تخمینی» even when BBT or LH confirmed it, the regularity pill reads the old overview
enum (v6 values make it disappear), and the card mixes engine and browser ranges (gap 02c D.2, D.5).

## Scope
1. Rows from `today_ui.prediction_card` when present: period `{start, end}`, PMS, fertile display window (hidden when
   null), ovulation with a source label (`bbt_confirmed` «تأیید با دمای پایه», `lh_positive` «بر اساس تست LH»,
   `calendar` «تخمینی»), cycle length and the regularity badge (`regular`, `irregular`, `insufficient_data` — calm
   wording, no "abnormal").
2. The overview entity maps both the v6 and the old regularity enums to one union.
3. Laravel payload → today's card (fallback path kept).

## Tests
- Card mapper: each ovulation source; null fertile window; each regularity value; Laravel payload unchanged.
- Visual QA light/dark (three states) under `docs/engine-v6/qa/`.

## Out of scope
- Suppression texts (T-M9-44).

## Acceptance
- Verify passes.
