---
id: L9-05
title: Test coverage + static analysis to level 8
milestone: L9
type: quality
status: todo
depends_on: [L9-04]
parallel_group: L9-D
touches: [tests,phpstan.neon,app]
skills: []
verify: composer verify
---

# L9-05 — Test coverage + static analysis to level 8

## Scope
- Feature test for every public route (200/301/404 as expected, head tags present, JSON-LD parses, no external URLs
  in HTML), every form (happy + validation + spam), every Filament resource (list/create/edit/authorization).
- Raise Larastan to level 8 (or max feasible, documented), fix findings; coverage report (pcov/xdebug if available)
  with a floor of 80% for `app/Domain` and `app/Support`.

## Acceptance
- `composer verify` green at the new levels; coverage numbers in PROGRESS.
