---
id: L0-08
title: Fidelity screenshot + diff tool (design HTML vs Laravel route)
milestone: L0
type: quality
status: done
depends_on: [L0-05]
parallel_group: L0-D
touches: [tools/shot.mjs,tools/README.md]
skills: []
verify: node tools/shot.mjs --help
---

# L0-08 — Fidelity screenshot + diff tool (design HTML vs Laravel route)

## Why
"Convert completely to Blade" needs proof: each converted page must look like its design page on phone and desktop.

## Scope
- `node tools/shot.mjs --design cycle.html --route /cycle [--widths 390,768,1440] [--out docs/qa/<task>]`:
  headless Chrome over CDP (no npm deps, Node ≥ 22 WebSocket; Chrome at `/Applications/Google Chrome.app`, override
  with `CHROME_PATH`), full-page PNGs of `design/html/<file>` (file://) and `http://127.0.0.1:8000<route>`, plus a
  per-width pixel-diff percentage and a diff PNG (small pure-JS PNG decode/compare, or `pngjs`+`pixelmatch` dev deps).
- `--all` reads the URL map from `docs/AUDIT.md` (or a JSON emitted next to it) and runs every converted page.
- Also records console errors, failed requests, and **any request to a non-local origin** (fails the run — enforces
  "no externals").
- `tools/README.md` documents usage and thresholds (target diff < 3% per width; differences explained in the task
  PROGRESS entry when above).

## Out of scope
- Lighthouse (L9-02).

## Acceptance
- Running it against the default Laravel page and `design/html/index.html` produces both shots and a diff number.
