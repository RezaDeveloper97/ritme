---
id: B-N8-01
title: Courses domain — instructors, courses, lessons, groups, phone-based access
milestone: N8
type: backend
status: done
depends_on: [B-N7-10]
parallel_group: N8-A
touches: [backend-go/db,backend-go/internal/learning,backend-go/internal/auth,backend-go/api]
skills: [new-endpoint]
verify: cd backend-go && go vet ./... && go test ./... && make lint
---

# B-N8-01 — Courses domain — instructors, courses, lessons, groups, phone-based access

## Why
Instructors (e.g. midwives) sell/grant courses to their students by phone number.

## Scope
- Instructor role on a user (admin-approved); courses → chapters → lessons (video/audio/pdf) + standalone content; draft/published; scheduled chapter unlock; groups; access grants by phone (pending until that phone signs up → auto-granted at OTP signup), duration unlimited/30/90/until date; progress (per lesson position, % complete); notifications «دوره برایت باز شد» + SMS via adapter.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Pending-phone grant activates on signup (test)
- `verify` green
