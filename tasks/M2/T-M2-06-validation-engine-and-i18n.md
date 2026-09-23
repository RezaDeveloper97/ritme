---
id: T-M2-06
title: Laravel-compatible validation engine and i18n (languages, locale, translations)
milestone: M2
type: backend
status: done
depends_on: [T-M2-03, T-M2-04]
parallel_group: M2-C
touches: [backend-go/internal/platform/validation, backend-go/internal/i18n, backend-go/resources/translations, backend-go/db/queries/i18n, backend-go/resources/lang]
skills: []
verify: cd backend-go && go test ./internal/platform/validation/... ./internal/i18n/... && make test-int PKG=./internal/i18n/...
---

# T-M2-06 — Laravel-compatible validation engine and i18n

## Why
Validation error bodies and messages are part of the contract (framework 422 family, `lang/fa/validation.php`,
`__('profile.*')`), and every endpoint resolves a locale. See api-inventory §2–§3, domain-inventory §2 (Translatable)
and §4.5, infra inventory §7.

## Scope
1. `platform/validation`: a small rule engine that reproduces the Laravel rules actually used in
   `app/Http/Requests/**` and inline `validate()` calls (inventory them first: required, nullable, sometimes,
   required_if, string, integer, numeric, boolean, array, in, regex, size, min/max (string/number/array semantics),
   between, date, date_format, before/after(_or_equal) incl. `today`, after_or_equal:<field>, email, etc.).
   Includes TrimStrings + ConvertEmptyStringsToNull input normalisation, `:attribute` humanisation
   (`log_date` → "log date", custom attribute names from the lang files), message lookup fa/en with Laravel
   fallback, error ordering identical to Laravel (rule order per field, field order of the rules array).
2. `resources/lang`: convert `backend/lang/{fa,en}/*.php` (+ Laravel's framework en validation defaults) to JSON once
   (script committed, output committed); `trans(key, params, locale)` helper.
3. `i18n.Registry` (= LanguageRegistry): active languages from DB, default language, BOOTSTRAP fa/en fallback,
   cached in Redis `ritme-go:languages.registry` with explicit `Flush()`; `Resolve()` (Accept-Language parsing:
   split `,`, strip `;q`, lowercase, `_`→`-`, exact or base match).
4. Locale middleware: `?locale=` → `Accept-Language` → default; stores the locale on the Fiber context. Helpers for the
   per-endpoint fa/en clamps (e.g. info, phase-content default fa, pregnancy content default en).
5. `Translatable`: `Pick(json, locale)` (requested → default → first non-empty), `Clean()`, plus the special
   `locale→fa→en` pick used by PhaseContent / home sections and the "whole object when missing" behaviour of
   PregnancyWeeklyContent (documented per call site).
6. `TranslationStore`: reads seed bundles (`backend/resources/translations/{fa,en}` copied into
   `backend-go/resources/translations`) deep-merged with live files under `STORAGE_PATH/app/translations/<code>/`,
   default-language fill.

## Out of scope
The `/languages` endpoints (T-M2-10). Admin language provisioning (T-M2-21).

## Acceptance
- Golden tests: for each FormRequest / inline rule set, a table of invalid inputs whose `errors` + summary `message`
  equal Laravel's (captured with the contract env; store as testdata).
- Registry and Translatable tests incl. a 3rd language (`ar`) and a missing-locale fallback.
- TranslationStore bundle for `fa`, `en`, `ar` equals `GET /api/v1/languages/{code}/messages` goldens.
