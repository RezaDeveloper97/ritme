---
id: T-M8-03
title: Copy batch 1 — core, shell, auth, onboarding, profile (pilot)
milestone: M8
type: frontend
status: todo
depends_on: [T-M8-02]
parallel_group: M8-B
touches: [frontend/messages/fa/common.json,frontend/messages/fa/nav.json,frontend/messages/fa/auth.json,frontend/messages/fa/welcome.json,frontend/messages/fa/onboarding.json,frontend/messages/fa/account.json,frontend/messages/fa/profile.json,frontend/messages/fa/profile-edit.json,frontend/messages/fa/profile-info.json,frontend/messages/fa/pwa.json,frontend/messages/fa/web-shell.json,frontend/messages/fa/notifications.json,frontend/messages/fa/banners.json,frontend/messages/fa/search.json,frontend/messages/fa/day-tasks.json,frontend/messages/fa/challenge.json,frontend/messages/fa/lab-consent.json,backend-go/resources/translations/fa/common.json,backend-go/resources/translations/fa/nav.json,backend-go/resources/translations/fa/auth.json,backend-go/resources/translations/fa/welcome.json,backend-go/resources/translations/fa/onboarding.json,backend-go/resources/translations/fa/account.json,backend-go/resources/translations/fa/profile.json,backend-go/resources/translations/fa/profile-edit.json,backend-go/resources/translations/fa/profile-info.json,backend-go/resources/translations/fa/pwa.json,backend-go/resources/translations/fa/web-shell.json,backend-go/resources/translations/fa/notifications.json,backend-go/resources/translations/fa/banners.json,backend-go/resources/translations/fa/search.json,backend-go/resources/translations/fa/day-tasks.json,backend-go/resources/translations/fa/challenge.json,backend-go/resources/translations/fa/lab-consent.json]
skills: [persian-writing:persian-writing]
verify: node frontend/scripts/copy-fa/check.mjs common nav auth welcome onboarding account profile profile-edit profile-info pwa web-shell notifications banners search day-tasks challenge lab-consent && cd frontend && npm run test && cd ../backend-go && go test ./internal/i18n/...
---

# T-M8-03 — Copy batch 1 — core, shell, auth, onboarding, profile (pilot)
**Model:** sonnet subagent rewrites; main session (opus) reviews. Namespaces: `common`, `nav`, `auth`, `welcome`, `onboarding`, `account`, `profile`, `profile-edit`, `profile-info`, `pwa`, `web-shell`, `notifications`, `banners`, `search`, `day-tasks`, `challenge`, `lab-consent`.

## Why
Part of the M8 Persian copy pass (user request 2026-10-18): every user-app string rewritten with the
`persian-writing` skill to STYLE.md (written-but-warm, «تو»). See `docs/copy-fa/INVENTORY.md` for this batch.

## Notes for this batch
Pilot batch: run it **alone** first. The main session compares the result with STYLE.md and amends STYLE.md /
GLOSSARY.md (or the checker) with every lesson before the parallel batches start. `lab-consent` and the
consent/privacy keys in `auth`/`onboarding` are legal-ish: keep every obligation and right exactly.

## Scope
- Rewrite `frontend/messages/fa/<ns>.json` for: common nav auth welcome onboarding account profile profile-edit profile-info pwa web-shell notifications banners search day-tasks challenge lab-consent; mirror to `backend-go/resources/translations/fa/`.

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
- Read the **full** `git diff` of the fa files. Reject and re-request any change that alters meaning,
  drops a hedge, changes a number, or breaks register. Spot-check 2 screens in fa on the local app
  (`local-dev` skill) for overflow of rewritten buttons/titles.

## Out of scope
- `en` and other locales, keys, components (a hardcoded string found in code → note it for T-M8-14),
  backend-go Go code/lang files other than the mirrored translations.

## Acceptance
- `check.mjs common nav auth welcome onboarding account profile profile-edit profile-info pwa web-shell notifications banners search day-tasks challenge lab-consent` → 0 errors; every remaining warning justified in PROGRESS.md.
- Reviewer read the full diff; no meaning/number/hedge change; register uniform per STYLE.md.
- `npm run test` (frontend) green; Go `internal/i18n` tests green (translations are embedded).
