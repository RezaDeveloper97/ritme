---
id: T-M2-31
title: Follow-ups from T-M2-30 — BMI data migration, article slugs, translation seed resync
milestone: M2
type: backend
status: done
depends_on: [T-M2-30]
parallel_group: M2-J
touches: [backend-go/db/migrations,backend-go/resources/translations,backend-go/internal/i18n/testdata,backend-go/contract,frontend/src/screens/home,frontend/src/screens/article,frontend/src/screens/articles, frontend/src/entities/article, frontend/src/app/message-scopes.ts,docs/go-migration]
skills: [verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make contract ROUTES=all && cd ../frontend && npm run typecheck && npm run lint && npm run test
---

# T-M2-31 — Follow-ups from T-M2-30 — BMI data migration, article slugs, translation seed resync

## Why
Open items from T-M2-30 (tasks/PROGRESS.md § T-M2-30).

## Scope
1. Goose data migration (next number after 00005): update `message_contents` rows for `bmi_message` `normal` and
   `obese` in `fa` from "Ritme" to «ریتمی» **only where the text still equals the old seeded default** (admin edits
   are kept). Down = reverse with the same guard. Integration test on the docker test stack.
2. Raw article category slugs on the home article rail (`screens/home`) and the related-articles rail
   (`screens/article`) → reuse `articleCategoryLabel` (`screens/articles/model/category.ts`); if FSD forbids a
   screen→screen import, move the helper down to `entities/article` and update the existing import.
3. Resync the backend translation seed `backend-go/resources/translations/<code>/` with `frontend/messages/{fa,en}`
   (CLAUDE.md §6.4): add missing namespaces (care, checkups, fertility, pregnancy v2, …) and update drifted ones
   (home, profile, pwa, articles). Re-record `internal/i18n/testdata/messages_*.json` and the affected `public`
   contract goldens / allowlist with a note in `docs/go-migration/deviations.md` if goldens diverge from Laravel.

## Out of scope
- Running the migration on stage/prod (happens on the next deploy; prod still Laravel until T-M2-26).

## Acceptance
- Migration idempotent + guarded, tested; slugs translated everywhere; seed equals frontend messages (a test or
  script proves it); verify green, contract green.
