# Canvas-build progress

Per-task notes (what shipped, commands/env/migrations, open items). Newest at the bottom.

## CB-CORE-01 — Canvas-v1 → bloom → code map

- Shipped `docs/canvas-build/README.md`: IA rules, board map (93 snapshot boards → bloom task / existing code / NEW + CB
  tasks), primitive gaps vs B-N1-03, token map (board hex → N&B tokens light+dark), gap list, conflicts vs DECISIONS.
  Rendered `docs/qa/canvas/boards/IA_Map.png`, `IA_Nav.png`.
- Findings: snapshot has 93 of 154 canvas boards (61 `nbd_` dark twins not snapshotted); no `(app)` route group exists
  yet (CB `touches` prefix provisional, B-N1-04 decides); Me = bloom `/profile`.
- CB task edits (scope/deps/touches): CB-CORE-02 (final primitive list, +5 boards), CB-NAV-03 (mode chip, «برای امروز»,
  «دسترسی سریع», + sheet rows, teen Services rule; +touches home/quick-access), CB-MENO-01 (reuse B-N2-01 profile
  columns, no menopause_profiles; +B-N2-01), CB-MENO-08 (menopause tab → score; +bottom-nav, +B-N1-04), CB-IVF-02 (IVF
  sub-mode nav «درمان»; +bottom-nav/home, +B-N1-04), CB-LOSS-02 (entry = bloom calm exit), CB-CONTRA-01 (one flag/one
  pill reminder with B-N1-09/B-N2-03; +B-N1-09), CB-PRIV-01 (reuse B-N1-11 neutral text; icon row dropped), CB-REC-05
  (no insurer access), CB-DIR-01 (online mode = request), CB-DIR-09 / CB-SHOP-08 (routes `/profile/bookings`,
  `/profile/orders`). `bash roadmap/bin/next.sh --check` → ok. INDEX.md not regenerated (titles unchanged).

TODO (ask user):
- [ ] Snapshot the 61 missing `nbd_` dark boards so QA can compare dark screens to a render? (default: tokens only)
- [ ] Plus vs bloom gates: menopause report share link (B-N6-04) and analysis patterns (B-N3-07) — free for canvas modes (DECISIONS #11) or keep bloom's Plus gate?
- [ ] Today «دسترسی سریع» persistence: local only (default) or a server preference?
- [ ] Tell bloom B-N10-01 to drop the shop link-out purchase path + partner «سفارش‌ها» row (DECISIONS #15)? CB-SHOP-01 depends on B-N10-03.
- [ ] Desktop web store badges (W_Dir_Booked, W_Shop_Done): link the Android WebView shell listing or hide?
- [ ] Directory cooperation terms («کارمزد یا اشتراک؛ هنوز تعیین نشده»): admin-editable placeholder OK?
- [ ] CB-NAV-03 waits on B-N7-01 (N7) + B-N3-03 — split into shell rules now / Services order after N7 (DECISIONS #3 "IA first")?

## CB-CORE-06 — Neshan map wrapper
- Shipped `@/shared/ui/map` (`NeshanMap`: dynamic, ssr:false; pins via portals, `top`/`controls`/`card` slots, "search this area", tap-only my-location, day/night Neshan styles from `data-theme`) and `shared/config/map.ts` (`resolveNeshanKey`, pinned SDK neshan-sdk 1.1.5 / mapbox-gl 1.13.2). No key / load failure → renders nothing + `onUnavailable`, so screens fall back to the list.
- New env: `NEXT_PUBLIC_NESHAN_KEY` (build-time). Contract + CSP needs in `docs/canvas-build/map.md`.
- Verify: typecheck, lint, fsd:lint, lint:styles, lint:dark, 711 tests, build — green. No screenshots (visible only in CB-DIR-07 / CB-INS-06, fidelity checked there).
- CSP change moved into CB-DIR-07 Scope (next.config.ts added to its touches).
- TODO (ask user): a Neshan *web* map key restricted to the app domains, supplied at build time on stage/prod.
- TODO (ask user): Neshan has no custom Night & Bloom style — uses Neshan's own day/night vector styles. Default centre = Tehran.

## CB-CORE-03 — Admin-editable content catalog
- Migration `00009_catalog_items` (+ Laravel schema twin `2026_09_30_000001`); `catalog_items(group, code, sort_order, is_active, audiences JSON, title/body/meta JSON i18n, needs_review)`.
- Routes: `GET /api/v1/catalog/:group?audience=` (auth:api, Redis cache per group, 10 min, flushed on admin write); admin CRUD `/api/admin/v1/catalog[/:group[/:id]]` (editor+, CSRF, audit). OpenAPI tag `Catalog`; deviation D-31.
- Seed convention per epic in `docs/canvas-build/catalog.md` (goose data migration `000NN_catalog_<group>.sql` + Laravel `insertOrIgnore` twin).
- Also touched `backend-go/sqlc.yaml` (catalog package) — required by sqlc; regenerated models.go across packages.
- Verify: sqlc, vet, unit + int tests, golangci-lint 0 issues, schema-diff OK (48 tables).
- TODO (ask user): catalog read is behind auth — should FAQ-type groups be public before login?
- Open: `admin-api.md` not updated (catalog admin endpoints documented in catalog.md §3); no bulk reorder endpoint (add in CB-CORE-04 if needed).

## CB-CORE-04 — admin-web: catalog editor
- `/catalog` (groups + counts, open new group), `/catalog/[group]` (search, status filter, drag/↑↓ reorder, active toggle, needs-review badge, audience chips), new/edit form (fa/en TranslatableField, audiences, JSON meta with per-group hints, 422 mapped to fields). Nav item under Content (editor+).
- Verify: typecheck, lint, fsd:lint, 109 tests, build — green. Screenshots `docs/qa/canvas/core/CB-CORE-04/` (fa/en, light/dark, 390px) — ✔.
- Follow-up CB-CORE-03b: atomic bulk reorder endpoint (today: one partial PUT per moved row) + duplicate-code 422 says «کد تایید» (shared OTP attribute label).
- Open: meta hints for `missed_pill_rules`, `pelvic_levels`, `_alerts`, `_score_items` are proposals in `admin-web/src/screens/catalog/lib/hints.ts` — CB-CONTRA-01 / CB-PELV-01 / CB-MENO-04 confirm or adjust.
