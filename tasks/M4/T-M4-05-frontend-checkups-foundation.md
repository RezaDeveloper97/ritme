---
id: T-M4-05
title: Frontend — checkups entity, features, tokens, icons, i18n and on-device attachments
milestone: M4
type: frontend
status: done
depends_on: []
parallel_group: M4-A
touches: [frontend/src/entities/checkup, frontend/src/features/record-checkup, frontend/src/features/manage-custom-checkup, frontend/src/shared/lib/local-files, frontend/src/app/globals.css, frontend/src/shared/ui/Icon.tsx, frontend/messages/fa, frontend/messages/en, frontend/src/shared/i18n, frontend/src/app/message-scopes.ts]
skills: [new-fsd-slice]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test
---

# T-M4-05 — Frontend — checkups entity, features, tokens, icons, i18n and on-device attachments

## Why
Foundation for every M4 screen; can start before the backend by coding against docs/checkups/README.md.

## Scope
1. `entities/checkup`: types, zod schemas (unknown status/tone → safe fallback), query hooks + `checkupKeys`
   (list, home, detail, records, preview-next).
2. `features/record-checkup` (create/update/delete record; invalidates list/home/detail/records),
   `features/manage-custom-checkup`, settings toggle.
3. `shared/lib/local-files`: IndexedDB store for report photos/PDFs keyed by record id (put/get/delete/list, size
   cap, graceful when storage is unavailable). Nothing is uploaded — privacy promise in the MarkDone copy.
4. Tokens for the status palette in the README (both `:root` and `[data-theme="dark"]`); reuse M3 tokens where
   they already exist (coordinate: if T-M3-04 is done, extend, don't duplicate).
5. Icons from the artboards: ribbon (breast), shield-check, flask, tooth, stetho (exists), camera, file, history,
   filter-lines, export.
6. i18n namespace `checkups` (fa + en) with all copy from the six artboards.

## Acceptance
- Parser + local-files unit tests; `lint:dark` and `lint:styles` green; no hex outside globals.css.
