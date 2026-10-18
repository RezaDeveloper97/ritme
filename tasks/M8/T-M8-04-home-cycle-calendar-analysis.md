---
id: T-M8-04
title: Copy batch 2 — home, cycle, calendar, phase details, analysis
milestone: M8
type: frontend
status: todo
depends_on: [T-M8-03]
parallel_group: M8-C
touches: [frontend/messages/fa/home.json,frontend/messages/fa/cycle.json,frontend/messages/fa/calendar.json,frontend/messages/fa/phase-details.json,frontend/messages/fa/analysis.json,backend-go/resources/translations/fa/home.json,backend-go/resources/translations/fa/cycle.json,backend-go/resources/translations/fa/calendar.json,backend-go/resources/translations/fa/phase-details.json,backend-go/resources/translations/fa/analysis.json]
skills: [persian-writing:persian-writing]
verify: node frontend/scripts/copy-fa/check.mjs home cycle calendar phase-details analysis && cd frontend && npm run test && cd ../backend-go && go test ./internal/i18n/... -skip TestBundle_MatchesLaravelGoldens
---

# T-M8-04 — Copy batch 2 — home, cycle, calendar, phase details, analysis
**Model:** sonnet subagent rewrites; main session (opus) reviews. Namespaces: `home`, `cycle`, `calendar`, `phase-details`, `analysis`.

## Why
Part of the M8 Persian copy pass (user request 2026-10-18): every user-app string rewritten with the
`persian-writing` skill to STYLE.md (written-but-warm, «تو»). See `docs/copy-fa/INVENTORY.md` for this batch.

## Notes for this batch
Highest-traffic screens. Phase descriptions and analysis insights are health explanations: keep every hedge.

## Scope
- Rewrite `frontend/messages/fa/<ns>.json` for: home cycle calendar phase-details analysis; mirror to `backend-go/resources/translations/fa/`.

## How to run this batch (every M8 rewrite batch)
1. Read `docs/copy-fa/STYLE.md` and `docs/copy-fa/GLOSSARY.md` fully; load the `persian-writing` skill and read
   its `references/writing-style.md` and `references/orthography.md`.
2. For each namespace: read the fa file **and** the en file (en is the meaning reference; fa may contain
   context en lacks — keep it). Find where keys render (`grep -rn "useTranslations('<ns>')" frontend/src`) when a
   string's context (button vs sentence, sheet title vs body) is unclear.
3. Rewrite values only. Never touch keys, ICU argument names/types, plural/select arms, tags, or `en`.
   Keep meaning exactly; for health text keep every number, unit, threshold, time window and hedge.
   Fix mixed register to STYLE.md, apply GLOSSARY.md terms, remove AI/bureaucratic tells, fix orthography.
   Do not lengthen buttons/tabs/chips. Do not "improve" a string that is already right — unchanged is fine.
4. Edit values surgically in the raw JSON text (never re-serialize a file). Then, in ONE shell call (env does not
   persist between calls): `PERSIAN_CLEANUP=<skill dir>/scripts/persian_cleanup.py node frontend/scripts/copy-fa/clean.mjs <ns…>`
   then `node frontend/scripts/copy-fa/sync.mjs <ns…>`. Do NOT touch `backend-go/internal/i18n/testdata/*` — the
   main session regenerates those once per round with
   `cd backend-go && go test ./internal/i18n -run MatchesLaravelGoldens -update` (parallel batches would collide).
5. `node frontend/scripts/copy-fa/check.mjs <ns…>` must have 0 errors; justify each remaining warning in the
   report (or fix it).
6. If a frontend test asserts an old Persian string, update the assertion to the new copy (list each one).
7. Report: per namespace — keys changed / total, the 10 most meaningful before→after pairs, any string you were
   unsure about (meaning ambiguous, possible medical nuance) **left unchanged and listed** for the reviewer.

## Review (main session, before `done`)
- Regenerate the i18n fixtures (`-update`, see step 4) and run `go test ./internal/i18n/...` without `-skip`.
- Read the **full** `git diff` of the fa files — phase descriptions and analysis insights line by line. Reject and re-request any change that alters meaning,
  drops a hedge, changes a number, or breaks register. Spot-check 2 screens in fa on the local app
  (`local-dev` skill) for overflow of rewritten buttons/titles.

## Out of scope
- `en` and other locales, keys, components (a hardcoded string found in code → note it for T-M8-14),
  backend-go Go code/lang files other than the mirrored translations.

## Acceptance
- `check.mjs home cycle calendar phase-details analysis` → 0 errors; every remaining warning justified in PROGRESS.md.
- Reviewer read the full diff; no meaning/number/hedge change; register uniform per STYLE.md.
- `npm run test` (frontend) green; Go `internal/i18n` tests green (translations are embedded).
