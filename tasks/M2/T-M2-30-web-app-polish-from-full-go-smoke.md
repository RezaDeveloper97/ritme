---
id: T-M2-30
title: Web app polish from full Go smoke
milestone: M2
type: frontend
status: done
depends_on: [T-M2-25]
parallel_group: M2-J
touches: [frontend/src/screens/calendar,frontend/src/screens/cycle,frontend/src/entities/health-log,frontend/src/entities/cycle,frontend/src/entities/article,frontend/src/screens/articles,frontend/src/features/log-period,frontend/src/widgets,frontend/messages, backend-go/contract/allowlist,backend-go/internal/profile,backend-go/internal/messages/content,backend-go/resources/translations,docs/go-migration]
skills: [verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run lint:styles && npm run lint:dark && npm run test && cd ../backend-go && go vet ./... && go test ./internal/profile/... ./internal/messages/...
---

# T-M2-30 — Web app polish from full Go smoke

## Why
The local full smoke on Go for T-M2-25 (docs/go-migration/stage-rollout-log.md § Local full smoke on Go) found 4
frontend/copy bugs (none Go-caused). File:line details are there.

## Scope
1. Hydration mismatch on `/calendar` and `/cycle`: queries gated on `isAuthenticated()` differ between SSR and client
   (DayLogSummary + health-log query, MyCyclesCard + cycle sections query). Gate on a client-mounted/hydrated flag
   (reuse an existing hook if there is one) so the first client render matches SSR.
2. Latin digits in fa: challenge «روز 1 چرخه» chip, calendar «روز 1 سیکل», period-editor badges 1–5 → locale
   number formatting (pass formatted numbers into messages, don't hardcode digits in JSON).
3. "Ritme" in Latin inside the fa BMI copy (backend-go profile/bmi.go and messages content defaults.json) → «ریتمی»
   in fa only; keep en. Update any Go test/golden that pins it (deliberate deviation from Laravel — note it in
   docs/go-migration as a decided deviation).
4. Article category chips and tags show raw slugs (`nutrition`, `cycle_science`) → map to translated labels
   (use the API's localized name if it returns one, else i18n keys in the articles namespace, fa + en).

## Out of scope
- Admin CSP for http thumbnails locally; vhost-admin (T-M2-26).

## Acceptance
- 1–4 fixed with unit tests where logic changed; verify green; no hydration warning on /calendar and /cycle in a
  local headless run; screenshots of the affected screens re-taken into docs/go-migration/screenshots/ (same names).
