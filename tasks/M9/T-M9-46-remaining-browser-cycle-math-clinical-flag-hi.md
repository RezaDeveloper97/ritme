---
id: T-M9-46
title: Remaining browser cycle math, clinical-flag hints and "normal" copy audit
milestone: M9
type: frontend
status: todo
depends_on: [T-M9-40, T-M9-32]
parallel_group: M9-C
touches: [frontend/src/screens/cycle, frontend/src/screens/phase-details, frontend/src/screens/companion-home/model/phase-bar.ts, frontend/src/screens/companion-home/model/phase-bar.test.ts, frontend/src/screens/analysis-cycle, frontend/src/entities/cycle/model/predictions.ts, frontend/src/entities/cycle/model/predictions.test.ts, frontend/src/entities/cycle/api/sections.ts, frontend/messages/fa/cycle.json, frontend/messages/en/cycle.json, frontend/messages/fa/analysis.json, frontend/messages/en/analysis.json, backend-go/resources/translations/fa/cycle.json, backend-go/resources/translations/en/cycle.json, backend-go/resources/translations/fa/analysis.json, backend-go/resources/translations/en/analysis.json]
skills: []
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run copy:check -- cycle analysis
---

# T-M9-46 — Remaining browser cycle math, clinical-flag hints and "normal" copy audit
**Model:** sonnet; the main session reviews flag copy (no diagnostic phrasing).

## Why
Four browser copies of cycle math disagree with v6: typical bands use O−5..O+1 and a 5-day PMS, the partner bar uses a
fixed 5-day period and O−4..O, the phase tabs invent ovulation at L−14, and `unknown` is mapped to luteal (gap 02c C,
D.1). `clinical_flags` exist but nothing shows them; §2.40 forbids calling 21–45 "normal".

## Scope
1. `screens/cycle/model/bars.ts`: window O−5..O, PMS last 3 days; `screens/companion-home/model/phase-bar.ts`: period
   length and ovulation from the engine (O−5..O), never fertile when `fertility_suppressed`;
   `screens/phase-details/model/tabs.ts`: ovulation from the engine or null (no L−14), fertile start from the period
   end; `entities/cycle/model/predictions.ts`: `unknown` → null (not luteal).
2. A shared 21/28/35-day table test (SPEC §2.36) run against all four functions.
3. `clinical_flags` → a calm "worth mentioning to your doctor" hint on `/cycle` and `/analysis/cycle` when frequency is
   frequent/infrequent, duration prolonged or absent is true (never shown as a diagnosis); teen ranges come from the
   backend.
4. Copy audit: no text calls 21–45 (or 21–35) "normal"; reword `analysis.json` key at line ~274 if needed.
5. Remove unused `useCycleSummarySection` / `useMyCyclesSection` hooks if grep shows no importer.
6. Laravel payloads keep working (each function has a fallback when v6 fields are absent).

## Tests
- The shared §2.36 table; updated `bars.test.ts`, `phase-bar.test.ts`, `tabs.test.ts`, `predictions.test.ts`.
- Flag hint view-model: each flag value; Laravel payload → no hint.

## Out of scope
- Home and calendar (T-M9-42…45).

## Acceptance
- Verify passes.
