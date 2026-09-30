---
id: B-N1-12
title: Privacy & security, support, about, legal
milestone: N1
type: fullstack
status: todo
depends_on: [B-N1-10]
parallel_group: N1-L
touches: [frontend/src/screens/privacy,frontend/src/screens/support,frontend/src/screens/about,frontend/src/features/app-lock,frontend/src/features/manage-account,backend-go/internal/profile,backend-go/internal/content,backend-go/api]
skills: [new-endpoint,verify-all,security-review]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N1-12 — Privacy & security, support, about, legal

## Why
Privacy is a headline promise of the design (intro slide 4).

## Design
- `docs/design/night-bloom/g-me-settings/nbl_Me_Privacy.dc.html` (+ `nbd_Me_Privacy`)
- `docs/design/night-bloom/g-me-settings/nbl_Me_Support.dc.html` (+ `nbd_Me_Support`)
- `docs/design/night-bloom/g-me-settings/nbl_Me_About.dc.html` (+ `nbd_Me_About`)
- `docs/design/night-bloom/g-me-settings/nbl_Me_Legal.dc.html` (+ `nbd_Me_Legal`)

## Scope
- App lock for the web app: 4–6 digit passcode (hashed, device-local) and optional WebAuthn platform authenticator; asks on open/resume after N minutes.
- Hide preview: blur overlay on `visibilitychange`/`pagehide`.
- Consents list (AI lab analysis, assistant uses profile, anonymous stats) stored server-side with timestamps.
- «دسترسی دیگران» section (companion + doctor share links) — rows appear when B-N4 / B-N6 ship.
- Export all data (JSON + PDF) and delete all data (existing endpoints, restyled).
- Support: chat link, report problem with screenshot (upload), FAQ from admin info-sections, phone/hours. About + Legal (3-line summary, TOC anchors) from admin content.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- App lock cannot be bypassed by reload/back navigation (tested)
- `verify` green
