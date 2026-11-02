---
id: T-M10-35b
title: Admin-web segment picker in the message rule condition editor
milestone: M10
type: frontend
status: done
depends_on: [T-M10-35]
parallel_group: M10-W
touches: [admin-web/src/screens/message-rules,admin-web/src/entities/segment,admin-web/src/screens/users,admin-web/messages/fa.json,admin-web/messages/en.json]
skills: []
verify: cd admin-web && npm run typecheck && npm run lint && npm run fsd:lint && npm run test && npm run build
---

# T-M10-35b — Admin-web segment picker in the message rule condition editor

## Why
Split from T-M10-35: its Acceptance asked for a segment picker in the message rule editor's audience section, but `screens/message-rules` was outside its touches.

## Scope
- Move the segments API/hook (`useSegments`, from `screens/users/api/segments.ts`) to `entities/segment` so two screens can share it (FSD: no screen-to-screen imports).
- Add a segment picker to the message rule editor's condition builder (`screens/message-rules`): choosing a segment writes a `segment` (set fact, T-M10-32) contains `<id>` condition; the rule audience stays `audience_percent` only (there is no segment audience field, by design).
- fa + en copy.

## Out of scope
- Standalone /segments page (segments stay a panel on /users).
- Backend changes (a segment audience field); segments target rules via the `segment` fact.

## Acceptance
- Rule editor can pick, show (by segment name) and clear a segment condition; saved rule round-trips it.
- Users screen still works from the moved entity.
- Vitest for the picker; light + dark screenshots in docs/engine-v6/qa/admin/.
- verify green. 

## Scope change (2026-11-02)
The rule audience has only `audience_percent`; segments reach rules through the `segment` fact. Picker moved to the condition builder; no backend change.
