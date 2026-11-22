---
id: T-M12-11d
title: Status banner seed and cycle daily card follow the M8 Persian copy (migration)
milestone: M12
type: backend
status: done
depends_on: [T-M12-11c,T-M12-13]
parallel_group: M12-TEST
touches: [backend-go/db/migrations,backend-go/internal/cycle/view,backend-go/contract/cases,backend-go/internal/messages]
skills: []
verify: cd backend-go && make test-int PKG="./internal/messages/... ./internal/cycle/..." && make contract ROUTES=cycle
---

# T-M12-11d — Status banner seed and cycle daily card follow the M8 Persian copy (migration)

**Model:** opus. Found in T-M12-11c (2026-11-18). Waits for T-M12-13 so two migrations are not written at once.

## Why
`TestFeedInt_BannerSeed` is red: the M8 copy pass (frontend only) rewrote `home.json` banner copy, but the
`status_banner` seed in `message_rules` (goose 00112) still has the old fa text, so users see the old wording.
`internal/cycle/view/daily_card.go` and some cycle contract goldens also still say «پایان پریود هنوز ثبت نشده».

## Scope
1. New goose migration: set `content.a.title`/`content.a.body` of `banner.end_missing` and `content.a.body` of
   `banner.overdue` to the M8 fa strings from `frontend/messages/fa/home.json` — only where the value still equals the
   00112 text (admin edits win). Down restores the old text. Data-only, no Laravel twin.
2. `daily_card.go`: same strings as the frontend copy (through translations, not literals, if possible); re-record the
   affected cycle goldens and explain each diff.

## Acceptance
- `TestFeedInt_BannerSeed` green; messages + cycle int green; `make contract ROUTES=cycle` green.
