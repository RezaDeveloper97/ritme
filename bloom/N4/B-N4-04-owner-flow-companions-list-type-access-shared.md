---
id: B-N4-04
title: Owner flow — companions list, type, access, shared children, invite, done
milestone: N4
type: frontend
status: done
depends_on: [B-N4-02,B-N1-10]
parallel_group: N4-D
touches: [frontend/src/screens/companion-*,frontend/src/entities/companion,frontend/src/features/invite-companion]
skills: [new-fsd-slice,verify-all]
verify: cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N4-04 — Owner flow — companions list, type, access, shared children, invite, done

## Why
Add a companion from Me.

## Design
- `docs/design/night-bloom/companion-family/nbl_Hamdam_List.dc.html` (+ `nbd_Hamdam_List`)
- `docs/design/night-bloom/companion-family/nbl_Hamdam_Type.dc.html` (+ `nbd_Hamdam_Type`)
- `docs/design/night-bloom/companion-family/nbl_Hamdam_Access.dc.html` (+ `nbd_Hamdam_Access`)
- `docs/design/night-bloom/companion-family/nbl_Hamdam_Children.dc.html` (+ `nbd_Hamdam_Children`)
- `docs/design/night-bloom/companion-family/nbl_Hamdam_Invite.dc.html` (+ `nbd_Hamdam_Invite`)
- `docs/design/night-bloom/companion-family/nbl_Hamdam_Done.dc.html` (+ `nbd_Hamdam_Done`)

## Scope
- List (family strip), Type (partner/spouse), Access (3-level per section), Children (spouse only), Invite (phone or code, copy/share), Done. Privacy screen lists active companions.

## Out of scope
- Android (android-shell/, application/, twa/) — never.
- Anything owned by another bloom task.

## Acceptance
- Light and dark both match the `nbl_` / `nbd_` artboards (full-page screenshots of both themes saved to `docs/qa/bloom/<task-id>/`, checked side by side)
- fa + en copy (frontend `messages/` + backend translation seed and i18n goldens in sync)
- Loading skeleton, empty and error states; touch targets ≥ 44px; RTL correct
- `verify` green
