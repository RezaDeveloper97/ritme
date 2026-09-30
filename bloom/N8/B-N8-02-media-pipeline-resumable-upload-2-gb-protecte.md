---
id: B-N8-02
title: Media pipeline — resumable upload (≤2 GB), protected streaming
milestone: N8
type: backend
status: todo
depends_on: [B-N8-01]
parallel_group: N8-B
touches: [backend-go/internal/media,backend-go/internal/learning,backend-go/api,docker-compose.yml,docker-compose.stage.yml,docker-compose.prod.yml]
skills: [new-endpoint,security-review]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N8-02 — Media pipeline — resumable upload (≤2 GB), protected streaming

## Why
Video lessons.

## Scope
- Chunked/resumable upload (tus-style), storage volume, signed short-lived URLs with range requests, per-user access check, optional HLS later (document). Size/type limits, virus-scan hook placeholder.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Resume after interruption test
- Unauthorised URL → 403
- `verify` green
