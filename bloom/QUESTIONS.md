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
