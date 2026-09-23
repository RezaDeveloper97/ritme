---
id: T-M2-21b
title: i18n lang loader reads admin-created languages from storage (D-04)
milestone: M2
type: backend
status: todo
depends_on: [T-M2-21]
parallel_group: M2-G
touches: [backend-go/internal/i18n/lang]
skills: []
verify: cd backend-go && go test ./internal/i18n/...
---

# T-M2-21b — i18n lang loader reads admin-created languages from storage (D-04)

## Why
T-M2-21's admin API writes `app/lang/<code>/*.json` for admin-created languages to the storage volume (D-04), but
`internal/i18n/lang` only reads the embedded `resources/lang` bundles, so validation/`trans()` messages for a new
language fall back to English.

## Scope
1. `internal/i18n/lang`: overlay `STORAGE_PATH/app/lang/<code>/*.json` on top of the embedded bundles (storage wins),
   loaded lazily per locale and reloadable (invalidate when the admin writes, or mtime check).
2. Unit tests with a temp dir: new locale served from storage; embedded fa/en unchanged; malformed file → ignored + logged.

## Acceptance
- A language created through `/api/admin/v1/languages` gets its own validation messages in Go without a restart.
- `go test ./internal/i18n/...` green.
