---
id: CB-CORE-05
title: Generalise bloom's encrypted file storage for all document kinds
epic: CORE
type: backend
status: done
depends_on: [B-N6-06]
parallel_group: CORE-B
touches: [backend-go/internal/files, backend-go/internal/labs, backend-go/db/migrations, backend/database/migrations, backend-go/api/openapi.yaml, docs/canvas-build/files.md]
skills: [new-endpoint]
boards: []
verify: cd backend-go && make sqlc && go vet ./... && go test ./internal/files/... && make test-int PKG=./internal/files/... && golangci-lint run && make schema-diff
---

# CB-CORE-05 — Generalise bloom's encrypted file storage for all document kinds

## Why
Record documents, claim documents, place photos/licences and product images need the storage B-N6-06 builds for labs.

## Scope
1. Extract the labs storage into `internal/files` (owner, purpose, mime, size, sha256, private/public, quota per purpose) without changing labs behaviour; signed short-lived download URLs; public variant for place/product images.
2. Purposes: record_document, claim_document, place_photo, place_licence, product_image.

## Out of scope
- Anything owned by another CB task; Android/native work (never); online payment (MVP).

## Acceptance
- Labs tests still green; round-trip tests per purpose.
- `verify` green.
