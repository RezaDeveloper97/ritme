---
id: B-N7-04
title: Doctor chat, e-prescription and video link adapter
milestone: N7
type: backend
status: todo
depends_on: [B-N7-03]
parallel_group: N7-D
touches: [backend-go/internal/telemed,backend-go/internal/care,backend-go/api]
skills: [new-endpoint,security-review]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N7-04 — Doctor chat, e-prescription and video link adapter

## Why
Pre-visit messaging and prescriptions.

## Scope
- Encrypted-at-rest messages user↔doctor, attachments (reports), prescription items → one-tap medication reminder; video link via adapter (fake room URL) active 10 min before; doctor side answers from admin-web (B-N7-08).

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Only the booked user and doctor can read a thread
- `verify` green
