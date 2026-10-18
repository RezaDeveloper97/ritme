---
id: T-M8-08
title: Copy batch 6 — health record, export, labs, home lab
milestone: M8
type: frontend
status: todo
depends_on: [T-M8-03]
parallel_group: M8-C
touches: [frontend/messages/fa/record.json,frontend/messages/fa/record-export.json,frontend/messages/fa/health-record.json,frontend/messages/fa/labs.json,frontend/messages/fa/home-lab.json,backend-go/resources/translations/fa/record.json,backend-go/resources/translations/fa/record-export.json,backend-go/resources/translations/fa/health-record.json,backend-go/resources/translations/fa/labs.json,backend-go/resources/translations/fa/home-lab.json]
skills: [persian-writing:persian-writing]
verify: node frontend/scripts/copy-fa/check.mjs record record-export health-record labs home-lab && cd frontend && npm run test && cd ../backend-go && go test ./internal/i18n/...
---

# T-M8-08 — Copy batch 6 — health record, export, labs, home lab
**Model:** sonnet subagent rewrites; main session (opus) reviews. Namespaces: `record`, `record-export`, `health-record`, `labs`, `home-lab`.

## Why
Part of the M8 Persian copy pass (user request 2026-10-18): every user-app string rewritten with the
`persian-writing` skill to STYLE.md (written-but-warm, «تو»). See `docs/copy-fa/INVENTORY.md` for this batch.

## Notes for this batch
Lab names/units stay exactly as written (HbA1c, TSH, mg/dL are LTR codes — never at the start of a line).

## Scope
- Rewrite `frontend/messages/fa/<ns>.json` for: record record-export health-record labs home-lab; mirror to `backend-go/resources/translations/fa/`.

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
4. `node frontend/scripts/copy-fa/clean.mjs <ns…>` then `node frontend/scripts/copy-fa/sync.mjs <ns…>`.
5. `node frontend/scripts/copy-fa/check.mjs <ns…>` must have 0 errors; justify each remaining warning in the
   report (or fix it).
6. If a frontend test asserts an old Persian string, update the assertion to the new copy (list each one).
7. Report: per namespace — keys changed / total, the 10 most meaningful before→after pairs, any string you were
   unsure about (meaning ambiguous, possible medical nuance) **left unchanged and listed** for the reviewer.

## Review (main session, before `done`)
- Read the **full** `git diff` of the fa files — every interpretation/range string in `labs` and `home-lab`. Reject and re-request any change that alters meaning,
  drops a hedge, changes a number, or breaks register. Spot-check 2 screens in fa on the local app
  (`local-dev` skill) for overflow of rewritten buttons/titles.

## Out of scope
- `en` and other locales, keys, components (a hardcoded string found in code → note it for T-M8-14),
  backend-go Go code/lang files other than the mirrored translations.

## Acceptance
- `check.mjs record record-export health-record labs home-lab` → 0 errors; every remaining warning justified in PROGRESS.md.
- Reviewer read the full diff; no meaning/number/hedge change; register uniform per STYLE.md.
- `npm run test` (frontend) green; Go `internal/i18n` tests green (translations are embedded).
