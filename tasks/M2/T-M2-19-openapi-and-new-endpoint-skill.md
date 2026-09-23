---
id: T-M2-19
title: OpenAPI spec for the Go API and new-endpoint skill for Go
milestone: M2
type: backend
status: todo
depends_on: [T-M2-10, T-M2-11, T-M2-12, T-M2-15, T-M2-16, T-M2-17, T-M2-18]
parallel_group: M2-I
touches: [backend-go/api/openapi.yaml, backend-go/internal/http/routes_docs.go, backend-go/internal/http/openapi_test.go, .claude/skills/new-endpoint/SKILL.md]
skills: [new-endpoint]
verify: cd backend-go && go test ./internal/http/... -run OpenAPI
---

# T-M2-19 — OpenAPI spec for the Go API and new-endpoint skill for Go

## Why
backend/CLAUDE.MD requires every endpoint documented; l5-swagger goes away with Laravel. And every future endpoint
must be written the Go way, so the `new-endpoint` skill must describe the Go conventions.

## Scope
1. `api/openapi.yaml` (OpenAPI 3.1), hand-maintained, starting from `backend/storage/api-docs/api-docs.json` but
   corrected with the real shapes from the contract goldens (decimal strings, date formats, both error families,
   201s). Served at `/docs` (Swagger UI from a CDN-free embedded bundle or Redoc) behind Basic auth
   (`SWAGGER_USER/PASSWORD`, fail closed outside local).
2. `openapi_test.go`: every registered Fiber route appears in the spec and vice versa; each golden body validates
   against its response schema.
3. Rewrite `.claude/skills/new-endpoint/SKILL.md` for Go: sqlc query → store → service → handler in
   `routes_<domain>.go` → validation rules → `httpx` envelope → contract case + golden (recorded from Go once
   Laravel is gone) → OpenAPI entry → fa/en content via `message_contents`/translations. Keep the Laravel
   instructions in a short "until T-M2-27" note.

## Out of scope
Admin API documentation (keep internal; optional section).

## Acceptance
- Spec ↔ routes test green; all goldens validate against the spec.
- `/docs` returns 401 without credentials and renders with them.
