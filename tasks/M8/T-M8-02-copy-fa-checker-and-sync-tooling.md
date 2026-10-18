---
id: T-M8-02
title: copy-fa tooling: ICU-safe checker, value-only cleanup and backend sync
milestone: M8
type: frontend
status: done
depends_on: [T-M8-01]
parallel_group: M8-A
touches: [frontend/scripts/copy-fa,frontend/package.json]
skills: [persian-writing:persian-writing]
verify: cd frontend && node --test scripts/copy-fa && cd .. && node frontend/scripts/copy-fa/check.mjs --all --baseline
---

# T-M8-02 — copy-fa tooling: ICU-safe checker, value-only cleanup and backend sync
**Model:** sonnet subagent; main session reviews the diff.

## Why
The skill's `persian_cleanup.py --edit` must NOT run on JSON directly: it turns `"..."` into «...», rewrites
ASCII digits, and would break ICU syntax. Rewrites need a mechanical safety net so a reviewer only judges
wording.

## Scope
1. `frontend/scripts/copy-fa/check.mjs <namespace…> | --all [--baseline] [--base <git-ref>]`:
   - **Structure gate (error):** same key tree as `git show <base>:frontend/messages/fa/<ns>.json` (default
     base `HEAD`) and as `frontend/messages/en/<ns>.json`'s key set; every value parses with `intl-messageformat`
     (already in node_modules); the set of ICU argument names, their types (`number`, `plural`, `select`, `date`)
     and plural/select arm keys is identical to the base; rich-text tags (`<b>…</b>`) identical.
   - **Sync gate (error):** `frontend/messages/fa/<ns>.json` byte-identical to
     `backend-go/resources/translations/fa/<ns>.json`.
   - **Lint (error):** Arabic ي/ك/ة-misuse, Arabic-Indic digits ٤٥٦, em/en dash inside Persian prose (a value that
     is exactly "—" as an empty-value placeholder is allowed), «می‌باشد», «لازم به ذکر», «در راستای», space before
     ، ؛ ؟, ASCII `"` quotes inside Persian prose, `می ` / `نمی ` without ZWNJ, `ها ` plural without ZWNJ where
     the dictionary says it should join.
   - **Lint (warn):** colloquial markers against STYLE.md register («میشه»، «می‌کنه»، «اومد»، «یه »، « رو »،
     «برات»، «داری؟» …), banned glossary variants from GLOSSARY.md (parse its rejected-variant column), Latin
     digits in prose, a value longer than 1.6× its base for keys whose name suggests a button/tab/chip
     (`cta`, `button`, `tab`, `chip`, `label`, `action`).
   - `--baseline` prints counts per namespace without failing on lint (used for INVENTORY and progress).
   - Output: `ns key: rule — value`; exit 1 on any error.
2. `frontend/scripts/copy-fa/clean.mjs <namespace…>`: walks every string value, protects ICU syntax
   (`{…}` blocks, `#` in plural arms, tags) behind sentinels, pipes the prose through the skill's
   `scripts/persian_cleanup.py --edit` (path passed via `--cleanup <path>` or env `PERSIAN_CLEANUP`; drop
   `fix_ezafe` so «ـه‌ی» style stays), restores, writes back with the file's existing JSON formatting
   (2-space indent, trailing newline, key order preserved).
3. `frontend/scripts/copy-fa/sync.mjs <namespace…> | --all`: copies fa namespaces to
   `backend-go/resources/translations/fa/`.
4. `node --test` unit tests for the ICU-protection round trip (plural with `#`, nested select, tags), each lint
   rule (one positive + one negative case), and the structure gate (renamed key, dropped plural arm, changed arg
   name must fail).
5. npm scripts: `copy:check`, `copy:clean`, `copy:sync`.

## Out of scope
- Rewriting any copy.

## Acceptance
- Tests green; `--all --baseline` runs over all 70 namespaces and the counts are appended to
  `docs/copy-fa/INVENTORY.md` (by the main session, outside this task's touches).
- `clean.mjs` on a copy of `home.json` changes no key, no ICU token, and the structure gate passes.
