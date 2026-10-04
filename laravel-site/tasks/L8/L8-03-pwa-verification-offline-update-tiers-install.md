---
id: L8-03
title: PWA verification (offline, update tiers, installability)
milestone: L8
type: quality
status: todo
depends_on: [L8-02]
parallel_group: L8-C
touches: [docs/qa/L8,tools/pwa-check.mjs]
skills: [pwa]
verify: node tools/pwa-check.mjs
---

# L8-03 — PWA verification (offline, update tiers, installability)

## Scope
- `tools/pwa-check.mjs` (CDP): load site, wait for SW, go offline, navigate visited + unvisited pages, assert
  expected responses; bump `min_build_id` via artisan tinker/command and assert forced screen; check manifest
  installability via `Page.getAppManifest` / `Page.getInstallabilityErrors`.
- Report in `docs/qa/L8/README.md` with screenshots (390 width).

## Acceptance
- All checks pass and are reproducible with one command.
