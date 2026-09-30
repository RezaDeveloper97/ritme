# Night & Bloom — open questions / TODOs for the user

Collected during the unattended `/bloom-task --parallel` run (started 2026-09-30). Each entry: task id, question,
default taken so work could continue. Review at the end of the run.

| # | Task | Question / missing input | Default taken meanwhile |
|---|---|---|---|
| 1 | B-N1-01 → B-N1-02 | Light artboards use two palettes: `#6E54F0`/`#231B3B` (16 newest hub screens) and `#7B61FF`/`#2F2F35` (the other 165, = today's palette). Which is canonical? | `#6E54F0` dialect (task spec + white text passes AA); B values treated as A in fidelity audits |
| 2 | B-N1-01 → B-N1-02 | Fresh-install theme: follow system (bloom decision) or light (current code deliberately has no `system`)? | `system` for fresh installs; a stored `light`/`dark` choice is kept |
| 3 | B-N1-01 → B-N3-08 | No artboard gives `/analysis` an entry point (the «تحلیل» nav tab appears only on older boards). Where does analysis live? | Under the mode tab: home cards «در تحلیل ببین»/«جزئیات», a ماه · سال · تحلیل segment on the calendar, and a Me row; nav keeps «خدمات» |
| 4 | B-N1-01 → B-N2-03 | No menopause/teen home artboards; what does the menopause «علائم» tab open? | `/analysis/symptoms`; menopause home = cycle home layout with menopause tiles; teen = simplified cycle home, no ads/shop/Plus upsell (full versions later via roadmap E02/E09) |
| 5 | B-N1-01 → B-N4-05 | `Hamdam_Home` (male companion) has no bottom nav drawn. | امروز · خدمات · من, no FAB, no mode tab |
| 6 | B-N1-02 | Acceptance greps `#FF6FAE` as "old palette", but the canonical artboards (`nbl_Cycle_Home`) still use it as the bloom/illustration pink. Keep it? | Kept as the light value of `--bloom` only; `lint:dark` bans it (and #7B61FF/#3DD6F3/#F2ECFF/#2F2F35/#131022…) everywhere else in `src/` + `offline.html` |
| 7 | B-N1-02 | Default canvas: the shell used to be white (`--surface`) with lavender only on some screens; artboards put every screen on `--page`. | `.app-shell` / mobile `.stage` now paint `--page`; cards stay `--surface` |
| 8 | (process) | A second Claude session (`/canvas-build`, `roadmap/`) edits the same working tree/branch concurrently (backend-go catalog migration 00009, `shared/ui/map`, `shared/config`). Bloom commits only stage bloom-owned paths; goose migration numbers may collide. | Bloom agents re-check the latest migration number right before creating one and never stage the other session files. |
| 9 | B-N1-03 | DateStrip: components.md says «today = solid `--brand-fill`», but the task's artboard (`Log_Sheet_Cycle`) draws the selected day as a `--brand` outline + 13% tint, and `Cycle_Home` uses yet another strip (dark ink cell, phase dots). Which is the primitive? | Primitive follows `Log_Sheet_Cycle` (selected = outline + tint, today = brand weekday, tone dot via `marker`); the home strip stays screen-specific (B-N1-06) |
| 10 | B-N1-03 | The dev-only `/dev/ui-kit` showcase uses sample Persian copy from the artboards, not i18n keys (it 404s in production builds). OK, or should it get a `uiKit` message namespace? | Untranslated sample copy, dev-only; route registered in `message-scopes.ts` with `['common']` |
