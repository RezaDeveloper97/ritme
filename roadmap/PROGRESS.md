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
