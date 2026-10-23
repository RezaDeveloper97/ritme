---
id: T-M9-31b
title: Expose engine and analysis revisions on the request context (auth guard)
milestone: M9
type: backend
status: done
depends_on: [T-M9-31]
parallel_group: M9-B
touches: [backend-go/internal/auth/guard.go,backend-go/internal/auth/revisions_int_test.go,docs/engine-v6/drafts/T-M9-31b-guard-revisions.diff]
skills: []
verify: cd backend-go && go vet ./internal/auth/... && go test -race ./internal/auth/... && make test-int PKG=./internal/auth/...
---

# T-M9-31b — Expose engine and analysis revisions on the request context (auth guard)
**Model:** opus. **Needs the user's OK before running:** it edits the auth guard (`guard.go`), which an automatic
safety check refused earlier in this session (T-M9-09c); do not bypass that.

## Why
T-M9-31 put `engine_rev` / `analysis_rev` on `passport.Session` (one auth query), but `guard.go` was outside its
touches, so routes cannot read them yet. T-M9-32…35 need `auth.CurrentRevisions(c)`.

## Scope
1. Apply `docs/engine-v6/drafts/T-M9-31b-guard-revisions.diff` (prepared and tested with `-overlay` by T-M9-31):
   `auth.Revisions` (alias of `passport.Revisions`), `RequireUser` stores the revisions in Locals,
   `auth.CurrentRevisions(c) (Revisions, bool)`, `Authenticate` keeps its signature; new `revisions_int_test.go`.
2. Delete the draft diff once applied.

## Acceptance
- verify green; `make test-int PKG=./internal/http/...` green; a route can read the revision with no extra query.

## Decision
User OK to edit `guard.go` (2026-10-23, this session).
