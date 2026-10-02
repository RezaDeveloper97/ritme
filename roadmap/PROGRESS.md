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

## CB-CORE-03b — Catalog bulk reorder endpoint and code attribute label
- `POST /api/admin/v1/catalog/:group/reorder {ids}` — full set, one transaction via new `SetCatalogSortOrder` query (only sort_order/updated_at), cache flush, audit `catalog_item.reorder`. admin-web uses it (one POST instead of per-row PUTs).
- Duplicate/missing catalog `code` now reads «کد آیتم» / "item code" via new lang group `resources/lang/{fa,en}/catalog.json`; shared OTP label untouched.
- Verify: vet, unit + int (Reorder, ReorderIsAtomic, CodeAttributeLabel), golangci-lint 0 issues, admin-web typecheck + 105 tests — green. Documented in docs/canvas-build/catalog.md §3.

## CB-CORE-02 — Extra UI primitives missing from bloom's set
- `frontend/src/shared/ui/nb`: SeverityScale, NumericScale (0–10 / 1–6, solid/soft), StepTimeline (numbered / dotted), CountdownRing (on bloom's ProgressRing) + `useTimer`, WeekDots, ProgressBar, RadioCardGroup (roving tabindex), SearchField, Checkbox; UrgentCard extended (`hotlines`, `actions`, `variant: card|note`) = DangerNote. 25 new tests.
- Verify: typecheck, lint, fsd:lint, lint:styles, lint:dark, build green; tests 774/775 — the one red (`message-scopes.test.ts` splash → welcome) is bloom B-N1-05 in flight, not this task.
- Fidelity: showcase light/dark `docs/qa/canvas/core/CB-CORE-02/` vs boards — ✔ (structure/states match; colours via tokens).
- Follow-up CB-CORE-02b: styles in side file `canvas-primitives.css` (not scanned by gates) → globals.css after bloom B-N1-04/05/13; ui-kit showcase entries; 0–10 scale touch width at 390 px; solid contrast.
- TODO (ask user / designer): SeverityScale colours (moderate = bloom, severe = danger) taken from boards — confirm.

## CB-PELV-01 — Pelvic floor backend: 8-week program, sessions, bladder diary
- Routes (Go-only, auth:api): `GET /api/v1/pelvic`, `POST|DELETE /api/v1/pelvic/program`, `POST /api/v1/pelvic/sessions`, `GET|PUT /api/v1/pelvic/diary/{date}` (uti_alert). OpenAPI tag `Pelvic`, contract `cases/pelvic.yaml` (21 goldens), deviation D-32.
- Migration `00010_pelvic_floor` (+ Laravel twin `2026_10_01_000001`): `pelvic_programs`, `pelvic_sessions`, `pelvic_bladder_logs` (user-scoped, cascade), plus catalog seed `pelvic_levels` / `pelvic_alerts` (doc: docs/canvas-build/pelvic.md).
- Verify: sqlc, vet, unit, int (16), golangci-lint 0, schema-diff OK (51 tables); full `go test ./...` green.
- CB-PELV-02 touches += backend-go/resources/translations (Go copy of frontend messages must be re-synced).
- TODO (ask user): include pelvic data in the account data export? Defaults: one session row/day (summed), stopping keeps history, level table needs clinical review.

## CB-CORE-02b — Canvas primitives into globals.css and ui-kit showcase
- Primitives' CSS moved into a delimited `CB-CORE-02 / CB-CORE-02b` block at the end of `globals.css` (side file removed) — now scanned by lint:styles (541 files ✔) and lint:dark ✔.
- `/dev/ui-kit`: «اجزای کانواس» section with every CB-CORE-02 primitive. Screenshots light/dark 390 px in `docs/qa/canvas/core/CB-CORE-02b/` — ✔.
- NumericScale: >7 steps wrap to rows (0–10 → 6+5, cells 54×48 px at 390); `solid` limited to brand/danger/success (AA with `--on-brand`).
- Verify: fsd:lint ✔, lint:styles ✔, lint:dark ✔, shared/ui tests 65/65 ✔; typecheck / lint / build / 1 test red **only** in bloom's in-flight B-N1-10/14/15 files (pregnancy*, checkup*, profile/account) — re-run full verify in the next frontend task after they land.
- TODO (ask user): 0–10 as two rows OK? `period` solid (3.8:1 light) dropped — use danger, or add a `--on-period` pair later.

## CB-CONTRA-01 — Contraception backend: method, pill pack, streak, long-acting reminders
- Routes (Go-only, auth:api, throttled writes): `GET /api/v1/contraception`, `PUT|DELETE /api/v1/contraception/method`, `POST /api/v1/contraception/pills`, `DELETE /api/v1/contraception/pills/{date}`. OpenAPI tag `Contraception`, contract `cases/contraception.yaml` (19 goldens), deviation D-36.
- Migration `00017_contraception` (+ Laravel twin `2026_10_01_000008`): `contraception_methods`, `contraception_pill_logs`, `contraception_reminders` (→ `reminders`, cascade) + catalog group `missed_pill_rules` (4 items, needs_review). 00016 intentionally unused (reserved by bloom B-N2-05, no migration).
- One switch / one schedule (C5): method save/stop drives bloom's `user_life_profiles.track_contraception`; the pill reminder is B-N1-09's `notification_preferences` `pill` category. Long-acting (IUD 6-week check, string check monthly, replacement, injection +12 w, implant, pack refill −5 d) are ordinary care `reminders` rows.
- Built in a worktree, landed on top of bloom B-N2-06 (sqlc regenerated). Verify: sqlc, vet, unit, int, golangci-lint 0 issues, schema-diff OK (67 tables), OpenAPI test, `make contract ROUTES=all` 1130 passed / 0 failed.
- Open: future push sender must skip break days and respect `track_contraception`; `PUT /profile/life-stage` turning the flag off doesn't disable the pill pref (profile code — follow-up); admin hints.ts add `methods` meta key; contraception not in `/profile/export`.
- TODO (ask user): days before setup = `untracked` (not missed) OK? export inclusion? clinical review of `missed_pill_rules` [needs clinical review]. Defaults: 28-day packs, streak includes break days, method reminders 09:00 Tehran, stop keeps pill history.

## CB-CONTRA-02 — Frontend: method setup + pill pack
- Routes `/contraception/setup` (method chips, pack type, start date, reminder time) and `/contraception` (today card, «یک قرص را جا انداختم» → /contraception/missed, pack grid taken/today/placebo, streak + next-pack tiles, refill row with packs-left sheet). Slices: `entities/contraception` (zod schema, queries, mutations writing the overview into the cache), `screens/contraception`, `screens/contraception-setup`; scopes `contraception`, `contraceptionSetup` (+ `profileMode*` load `contraception`).
- Mode screen (B-N2-03): switch ON → setup (save turns the flag on); OFF → `DELETE /contraception/method` (method + reminders removed, pill log kept) — closes CB-CONTRA-01's "flag off leaves pill reminder on" item. Manage row «روش و بسته قرص» while tracking.
- Task touches corrected: routes live at `app/[locale]/contraception` (no `(app)` group). Go copy of messages in `backend-go/resources/translations/{fa,en}/contraception.json`.
- Verify: typecheck, lint, fsd:lint, lint:styles (651), lint:dark, 939 tests, build — green. Fidelity `docs/qa/canvas/contra.md`: Setup ✔, Pill ~ (settings button in header for the unlabelled board link; legend says «تیک‌دار» not «سبز»; hormonal IUD `warm` tone).
- Open: `/contraception/other` + `/missed` links 404 until CB-CONTRA-03; no Home entry point yet (CB-CONTRA-04); i18n goldens need re-recording once bloom's message changes land.
- TODO (ask user): switch OFF deletes the method — OK? IUD default lifetime copper 10 y / hormonal 5 y [needs clinical review]; start date defaults to today.

## CB-CONTRA-03 — Frontend: missed-pill guide + other methods
- `/contraception/missed`: count chips → numbered steps, urgent rule as bloom's UrgentCard, general-guidance note — all from catalog `missed_pill_rules` (no hard-coded clinical copy); progestin pill shows `progestin_note`; non-pill users get an empty state.
- `/contraception/other`: only the user's own method — IUD (monthly string-check switch via `PUT /reminders/{id}`, replacement year, 6-week check «انجام شد» via `PUT /contraception/method followup_done`), injection (countdown + «امروز تزریق کردم» from 21 days before due), implant (date sheet → `replace_on`). No new endpoints.
- Entity: `missedGuide`, `methodReminder`, `useMissedPillRules`. Scopes `contraceptionMissed`, `contraceptionOther`. i18n goldens updated (contraception only).
- Verify: typecheck, lint, fsd:lint, lint:styles (669), lint:dark, 954 tests, build — green; `go test ./internal/i18n` ok. Fidelity `docs/qa/canvas/contra.md` both ✔ (board shows all three methods stacked; real data shows the user's own).
- TODO (ask user): «انجام شد» via followup_done (not appointments) OK? injection button from 21 days before? default missed chip «۱ قرص»? neutral note copy for injection/implant [needs clinical review].

## CB-CONTRA-04 — CONTRA QA
- Journey from the UI (mode switch on → setup → pill log/undo → missed → injection → switch off → restore) with DB/API checks after each step — all as specified; pill log survives stop.
- Reminders: nothing is *sent* yet (no push sender/cron in backend-go). Pill pref `pill` on (21:00) for pill methods, off otherwise; `internal/reminders.Plan` emits `send=true` / `category_off` accordingly. Injection/IUD/implant reminders exist as care `reminders` rows with correct dates (visible in GET /reminders and /care/appointments).
- Fix: next-pack tile date no longer wraps («22 October», «۳۰ اردیبهشت») — `.ctr-stat-num.is-long`.
- Final verdicts (`docs/qa/canvas/contra.md` § CB-CONTRA-04): Setup ~ (injection/implant icons → CB-CONTRA-04b), Pill ✔, Missed ✔, Other ✔. No ✘.
- Verify: typecheck, lint, fsd:lint, lint:styles (669), lint:dark, 954 tests, build — green.
- For the future push sender: skip pill on break days; pill time inside quiet hours currently defers to 08:00 (ask user). No Home entry point to /contraception yet (reached via /profile/mode) — ask user whether a home card is wanted.

## CB-CONTRA-04b — Syringe and implant icons for contraception methods
- `syringe` and `implant` glyphs added to `shared/ui/Icon.tsx` (paths from the board's tiles; the board's implant is a circle with rays, not a rod); `METHOD_LOOK` uses them for injection / implant (setup tiles, other-methods cards, pill screen).
- Verify: typecheck, lint, lint:styles, lint:dark, 954 tests — green. Setup verdict → ✔ (`docs/qa/canvas/contra/CB-CONTRA-04b/`).

## CB-NAV-01 — Global search API
- `GET /api/v1/search?q=&scope=all|mine|education|programs|services&limit=` (auth:api, Go-only, D-38). Groups in board order mine → programs → education → services; `total` + up to `limit` items (3 on all, 20 per scope); `shop` rejected (422).
- mine = caller's log insights (log-taxonomy labels for her mode; days + peak score in the current cycle or last 30 days), analysis links, her care medications/appointments. programs = contraception, pelvic. education = published articles. services = checkups/reminders screens + visible checkup types.
- Persian normalisation (ي/ك, alef/heh variants, digits, diacritics, ZWNJ) with tests. Query never logged, nothing cached; user-isolation int test. No migration, no new SQL.
- Verify: vet, golangci-lint 0, go test ./..., test-int search, OpenAPI test, `make contract ROUTES=all` (12 search goldens) — green.
- Open: contract fixture lacks checkup_types (services covered by int tests only); routes not yet built (/programs/pelvic CB-PELV-02, /analysis/* B-N3-08/09, /articles/{slug}) — CB-NAV-02 maps or hides them; add sources when condition programs (CB-COND-01), courses (B-N8), directory (B-N7/CB-DIR) ship.
- TODO (ask user): insight window = current cycle for cycle/ttc/teen, 30 days otherwise; response echoes raw query; keep pelvic link before CB-PELV-02.

## CB-MENO-01 — Menopause schema, taxonomy items and seeds
- Goose `00022_menopause` (+ Laravel twin `2026_10_01_000022`): `hot_flashes`, `menopause_scores` (Jalali month, 11 answers, domain subtotals 16/16/12), `treatment_items` (→ reminders, set null), `treatment_intakes`, `side_effect_logs`; `checkup_types.audiences` (JSON modes, NULL = everyone). Profile fields reuse bloom's `user_life_profiles.menopause_*`.
- Taxonomy (bloom B-N3-01, extended — no parallel log): menopause-only items anxiety/irritability/low_mood, bladder_symptoms/low_libido, param `bleeding.presence` (alert), mode category `menopause.triggers`; `taxonomy.MenopausePreset()` (board's 5 groups + bleeding + triggers). Labels in log-taxonomy.json (Go + frontend copies) and i18n goldens.
- Catalog (audience menopause, needs_review): `meno_score_items` 11, `meno_score_bands` 4, `meno_alerts` 7, `meno_tips` 14, `meno_checkup_groups` 4. 9 `meno_*` checkups seeded **inactive** → CB-MENO-01b (audience filter + activation). Doc `docs/canvas-build/menopause.md`.
- Verify: sqlc, vet, go test ./..., golangci-lint 0, schema-diff OK, contract all green, migrations round-trip int test.
- Pre-existing red int tests (not this task): internal/catalog admin tests assume empty catalog_items (broken since 00010/00017 seeds); internal/admin/languages, internal/admin/messages.
- TODO (ask user): Jalali score month; checkups inactive until CB-MENO-01b; eye/dental split; relaxation 70 min/week; «گرما» → warm_room. All 40 catalog rows + 9 checkups [needs clinical review].

## CB-COND-01 — Condition programs backend: enrolment, pain diary, PMDD, PBAC
- Routes (Go-only, auth:api, throttled writes; deviation D-40): `GET /api/v1/conditions`, `POST /conditions/enrolments`, `DELETE /conditions/enrolments/{program}`, `GET|PUT /conditions/pain/{date}`, `GET /conditions/pmdd/chart`, `GET|PUT /conditions/pmdd/{date}`, `GET|PUT /conditions/pbac/{date}`. OpenAPI tag `Conditions` (10 ops), contract `cases/conditions.yaml` (42 goldens).
- Goose `00023` (+ twin `2026_10_01_000023`): `condition_enrolments`, `condition_pain_entries`, `pmdd_entries`, `pbac_entries`. Pain locations/score/relief, slot-backed associated symptoms and PBAC clots are written to bloom's `health_log_entries` via `healthlog.Service.SaveDay` in the same tx — no parallel log.
- Catalog (needs_review): `condition_programs` 4, `pain_types` 4, `pain_associated` 5 (meta `log`), `pmdd_items` 6, `condition_alerts` 9 (crisis hotlines 1480/123).
- PMDD: late-luteal = last 10 days, follicular = days 4–10, complete cycle 7+4 rated days, ready = 2 cycles, luteal pattern at ≥1.3× rise. PBAC 1/5/20, clots 1/5, flooding 5, alert ≥100, 10-day window.
- Verify: sqlc, vet, go test ./..., unit + int conditions, golangci-lint 0, OpenAPI, contract all, schema-diff OK. Pre-existing red: internal/catalog admin tests.
- Open: pain diary rewrites every location with one score (overwrites per-location scores from the log sheet); board's finer pain locations not in bloom's taxonomy; admin hints for meta `log`/`logs`/`hotlines`; not in /profile/export; no docs/canvas-build/conditions.md.
- TODO (ask user): writes require enrolment; `associated: null` clears slot-backed symptoms; PMDD mean over rated items only; export inclusion. All clinical rules/copy [needs clinical review].

## CB-COND-06 — Nudges: heavy pain/bleeding → suggest program & doctor
- Engine `internal/messages/conditionnudges` (pure `Detect` rules + engine + handler): `heavy_pain` = pain ≥7 (score; level `severe` when unscored) on ≥2 days of the current cycle while not enrolled in endo → pain diary; `heavy_bleeding` = flow heavy/very_heavy on ≥2 days while not enrolled in heavy_bleeding → PBAC. Cycle window = last period start (cycle_histories / profile) to today, else last 28 days (or profile cycle 21–45). Modes cycle/ttc/teen only. Reads bloom's health_log_entries; nothing stored; recomputed per request.
- Copy: live `message_contents` group `condition_nudge` with per-field fallback to embedded fa/en (`needs_review` true while any fallback is used); not seeded (no migration). Doctor hook `DoctorDirectory` returns null until B-N7-02 — no dead link.
- Verify: vet, go test ./..., golangci-lint 0, int messages (17), contract all green.
- Follow-up CB-COND-06b: admin registry group + mount `GET /api/v1/messages/nudges` + OpenAPI/contract/deviation; CB-COND-02 shows the card.
- TODO (ask user): PBAC ≥100 should also trigger the bleeding nudge? show in teen mode (yes now)? thresholds [needs clinical review]; no dismiss/dedupe yet.

## CB-MENO-01b — Checkups audience filter and menopause checkup activation
- Checkups engine: shared types filtered by `audiences IS NULL OR JSON_CONTAINS(audiences, life_mode)` (mode via `enums.ResolveLifeMode`) in plan/list/detail queries; out-of-audience detail/record/settings → 404; `LoadPlan` still 3 queries. Search (`internal/search` VisibleCheckups) passes the user's mode.
- Admin checkup-types: `audiences` read/write (nullable list of the 6 modes, validated; `[]`/null → NULL; absent on update keeps value) + `options.audiences`. Docs: admin-api.md §12, menopause.md §6.
- Goose `00024_activate_menopause_checkups` (data only; no Laravel twin — schema-diff compares schema + counts) activates the 9 `meno_*` rows.
- Verify: sqlc, vet, go test ./..., golangci-lint 0, int checkups/admin checkups/search/migrations, schema-diff OK, contract all green. No contract case (fixture has no checkup_types; admin routes not in the runner) — int tests cover it.
- TODO (ask user): after a mode switch, records of now-hidden types stay in `/checkups/records` history (detail 404, own record update/delete still allowed) — OK?

## CB-MENO-04 — admin-web: menopause content
- Catalog hints (with «درج مثال» from the shipped seeds) for `meno_score_items`, `meno_score_bands`, `meno_alerts`, `meno_tips`, `meno_checkup_groups`; aligned shipped groups: `missed_pill_rules` (+`methods`, `pack_week`), condition groups (`condition_programs` logs, `pain_associated` log, `condition_alerts` severity/hotlines, `pain_types`, `pmdd_items`); `_score_items` hint key fixed `score_max` → `max`. Closes CB-CORE-04's "proposed hints" open item.
- Checkup types: `audiences` multi-select (empty = everyone), «نمایش برای» column, life-mode list filter (client-side; reorder disabled while filtered); toggling «فعال» keeps audiences.
- Log taxonomy is code (not exposed by bloom's admin) — nothing to build.
- Verify: typecheck, lint, fsd:lint, 123 tests, build — green. No screenshots: local admin login was blocked by the permission system in this session.
- TODO (ask user): filter «یائسگی» shows only explicitly targeted rows (shared under «همه») — or include shared? fa mode labels OK? allow local admin login for screenshots in CB-MENO-13.

## CB-COND-06b — Condition nudges route and admin registry
- `GET /api/v1/messages/nudges` (auth:api, localized; D-41) → `conditionnudges.Handlers.Index`; OpenAPI `getMessagesNudges`; contract `nudges` / `nudges_flow` / `nudges_anon` (11 Go-recorded goldens). Admin registry group `condition_nudge` (typed heavy_pain / heavy_bleeding; title/body/action/doctor_action; `{days}`), create/edit int test. fa `{days}` in Persian digits.
- Also fixed stale `TestMissingAndRegistry` count (9 → 10, red since `loss_exit`).
- Verify: vet, messages + admin/messages tests, int conditionnudges + http, golangci-lint 0, OpenAPI, contract all (1253 passed) — green.
- Open: a later heavy-flow day starts a new period → cycle window resets and earlier heavy days stop counting (by design; review). Frontend card = CB-COND-02.

## CB-MENO-02 — Menopause API: profile, today, hot flashes, score, patterns
- Routes (Go-only, auth:api, throttled writes; D-42; OpenAPI tag `Menopause`; contract `cases/menopause.yaml`, 37 goldens): `GET|PUT /api/v1/menopause/profile` (stage rule computed from bloom's `user_life_profiles.menopause_*`; `suggested_stage`, `needs_stage`, stage tip), `GET /menopause/today` (flashes + running timer, last night's sweats/sleep, latest score + band + delta + 6-month trend, ≤3 upcoming checkups, treatment adherence 7 days read-only, bleeding alert flag), `GET|POST /menopause/hot-flashes` + `POST /hot-flashes/{id}/stop` (1 h cap, 7-day backfill), `GET|POST /menopause/scores` (Jalali month, domains, band, delta, HRT note), `GET /menopause/patterns` (90-day φ via bloom's `analysis.Binary` wrapper; not_a_diagnosis / not_causal / needs_review).
- Contract fixture dump.sql now has the checkup + menopause tables (additions only). Doc `docs/canvas-build/menopause.md` §7.
- Verify (landed on top of CB-MENO-01b): sqlc, vet, go test ./..., golangci-lint 0, int menopause, contract all 1290 passed — green.
- TODO (ask user): bleeding alert also for older period logs within 30 days after switching to menopause? score delta vs previous filled questionnaire; upcoming checkups include never-done; pattern sentences in package lang. Clinical constants (12 months, 30-day bleeding window, night 22–06, patterns) [needs clinical review].

## CB-MENO-12 — Message engine: menopause tips & alerts
- `GET /api/v1/messages/menopause` (auth:api, localized; D-43; OpenAPI `getMessagesMenopause`; contract `menopause`, `menopause_flow`, `menopause_anon`, 11 goldens) → `{mode, stage, messages[{key, kind, priority, title, body, action, link, needs_review, data}]}`; non-menopause users get `[]`. Package `internal/messages/menomessages` (pure rules + engine + handler); facts from new `menopause.Service.Signals` (reuses stage rule, bleeding flag, score delta, checkups plan, treatment activeOn).
- Rules (constants, [needs clinical review]): `postmenopausal_bleeding` (high) → /menopause/alert; `checkup_overdue` → /checkups/{id}; `score_worsened` (≥4 vs previous, fresh within 2 Jalali months) → /menopause/score; `hrt_review` (review_on within 14 days) → /menopause/treatment; `checkup_due` (only when nothing overdue; never-done counts); stage tips from `meno_tips` (placement=home).
- Copy: bleeding + tips from catalog; other rules via new admin registry group `menopause_message` (typed, unseeded, per-field fallback to embedded fa/en). No migration (00025 unused).
- Verify: vet, go test ./..., int menomessages/menopause/admin messages/checkups/http, OpenAPI, contract — green at HEAD. Bloom's in-flight B-N2-11b (checkups timing for menopause) will turn `messages/menopause_flow` `timing_label` to null → re-record those 2 goldens after it lands.
- TODO (ask user): `checkup_due` fires for never-done checkups (almost always shown) — only re-due ones? no dismiss/dedupe; tips uncapped.

## CB-NAV-02 — Global search screen
- `/search` (flow, no bottom nav): field + cancel, scope chips (همه / از ثبت‌های تو / آموزش / خدمات / برنامه‌ها), grouped lists with «همه N» → scope, skeleton/too-short/error/no-results states, shop note («فروشگاه در خدمات › فروشگاه»). 300 ms debounce, TanStack Query, query never in the URL. Recent searches: localStorage (try/catch), max 6, clearable, stamped with user id and dropped for another account.
- Result routing whitelist (no 404 links): contraception, checkups (+self-exam, {id}), reminders, analysis/*, articles → article sheet; everything else hidden (e.g. /programs/pelvic until CB-PELV-02).
- Search button in the Today header of every mode (HomeHeader covers cycle/ttc/teen/postpartum/menopause; PregnancyPage Header). Search lives in the screen slice (an `entities/search` slice tripped steiger excessive-slicing; only consumer is the screen). Messages `search.json` (+ Go copy + i18n goldens).
- Verify: typecheck, lint, fsd:lint, lint:styles (727), lint:dark, build green; 1050 tests (on top of bloom batch faba825). Fidelity `docs/qa/canvas/nav.md` ✔ (chip «خدمات» instead of «پزشک» until doctors exist).
- Open: add /programs/pelvic, doctors (B-N7), courses (B-N8), condition programs to the whitelist when they ship; drop /analysis/* if B-N3-08 doesn't land.
- TODO (ask user): «خدمات» vs «پزشک» chip; cycle count on the analysis row needs CB-NAV-01 meta; on-device recents OK for privacy?

## CB-VOICE-01 — Voice parser coverage for canvas-v1 items
- `internal/ai`: `Candidate.Alternatives` (ambiguity, e.g. «بی‌حوصله» → sad / fatigue), vocab types `text` + `time`; Gemini prompt/parse updated; fake fixtures fa+en for menopause, pain diary, pill, pelvic; number words → digits; pain score → level.
- `POST /logs/voice` suggestions now carry `target` (log | hot_flash | pain_diary | pill | bladder) + `options[]`; diary items only when eligible (menopause mode / endo enrolment / pill method / everyone for bladder).
- New `POST /api/v1/logs/voice/commit` (auth + write throttle, not Plus-gated — no AI; within D-44's voice group): validates all items first, then writes through `menopause.Service.StartFlash`, `conditions.Service.SavePain`, `contraception.Service.LogPill`, `pelvic.Service.SaveDiary`. Log items still go through `PUT /logs/days`.
- Verify: sqlc, vet, go test ./..., golangci-lint 0, int voicelog, OpenAPI, contract all 1325 passed — green. No migration.
- Open: pain score >0 needs a location saved first (client: PUT /logs/days before commit); writes across 4 services not one transaction; default flash 3 min, start 03:00 / noon [needs clinical review]; frontend VOICE-02/03 to use commit + options; per-item quote not produced.
- TODO (ask user): bladder diary for everyone or only pelvic/menopause? commit also accept log items (one call)? Plus-gate commit? keep 3-min default?

## CB-MENO-05 — Frontend: menopause home + stage setup
- `/menopause/stage` (4 radio cards, Jalali last-period month picker, surgical, HRT → PUT /menopause/profile; «شروع» first time, then «ذخیره»; no bottom nav). All 4 stages re-tested end-to-end via the UI.
- Menopause home replaces bloom's minimal `MenopauseHome` under the shared Today header (search button kept): months-without-period hero + stage chip, quick actions (hot flash timer start/stop, log today → `?sheet=log`, bleeding/spotting), today stats, score card + 6-month sparkline + delta, 3 upcoming checkups, treatment card, `/messages/menopause` cards (bleeding via UrgentCard). Links to routes that don't exist yet (score details, treatment, doctor report, /menopause/alert) are hidden.
- `entities/menopause` (schema, queries, hot-flash timer). steiger `excessive-slicing` turned off for entities (21st slice) + `insignificant-slice` off for entities/menopause. Messages `menopause.json` (+ Go copy + i18n goldens).
- Verify: typecheck, lint, fsd:lint, lint:styles (792), lint:dark, 1064 tests, build — green. Fidelity `docs/qa/canvas/meno.md` (Stage, Home, Main) — no ✘. Test user 09120005055 (menopause; one HRT row inserted directly in ritme_dev).
- Open: mode switcher doesn't open the stage screen (screens/mode outside touches); bleeding tile has no log-sheet section preset; dead bloom keys `home.life.menopause.*` / `.meno-*` CSS; wire hot-flash tile → /menopause/hot-flash (CB-MENO-07), score details (08), treatment (10), report (11). Pre-existing: `/messages/daily` 400 in menopause mode.
- TODO (ask user): steiger rule vs slice groups; sparkline direction.
