# Night & Bloom — progress notes

One `## B-Nx-NN` section per finished task: what shipped, commands/env vars, migrations, open items.

## B-N1-01 — Design import audit

- Shipped `docs/night-bloom/` — `README.md` (summary + per-task reading list) linking `tokens.md` (light/dark colour
  table with proposed CSS variables + legacy aliases, gradients, starfield/glow, type, radii, shadows, spacing,
  contrast), `components.md` (inventory → artboards), `routes.md` (183 artboards → route/sheet, restyle vs NEW,
  owner task), `nav.md` (per-mode tabs, visibility, deviating boards), `gaps.md` (19 gaps with defaults). No code changed.
- Key findings: two light dialects (canonical `#6E54F0`); brand gradient gone (solid CTAs/FAB); dark primary fill is
  lavender with dark text (`--on-brand` flips); Lalezar used for display titles too (current subset font too small);
  no analysis entry point in the nav; 55 restyle / 119 NEW / 9 reference-only screens.
- Task scopes updated: B-N1-02, B-N1-03, B-N1-04, B-N1-14 (Design), B-N2-03, B-N3-08, B-N4-05, B-N8-03.
- Open: `bloom/QUESTIONS.md` #1–#5; B-N1-02 `touches` lacks `frontend/src/app/fonts` + `frontend/src/app/manifest.ts`
  and B-N1-04 `touches` names a non-existent `(app)` route group — adjust frontmatter when those tasks start.
