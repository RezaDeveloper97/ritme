# Web-home performance — query budgets and the fan-out (T-M12-30, finding P-05)

## 1. What the web home fires on a cold open

Cycle-mode user, engine v6, feed answering (so the legacy `/messages/daily` is not asked). Source: the hooks of
`frontend/src/screens/home/ui/HomePage.tsx` and the widgets it mounts. `loadtest/web_home.js` replays exactly this list.
Everything is a GET; TanStack Query dedupes the keys that several components share (banners, ads, life stage, feed).

**Wave 1** — `HomePage` (`HomePage.tsx:1574-1575`), before anything else is mounted:

| # | GET | Hook (source) |
|---|---|---|
| 1 | `/profile/life-stage` | `useLifeStage` — `entities/user/api/life-stage.ts:65` |
| 2 | `/messages/mode` | `useUserMode` — `entities/message/api/queries.ts:23` |

**Wave 2** — `CycleHome`, all at once once the mode is known (`HomePage.tsx:631-969` and the widgets at `:1312-1527`):

| # | GET | Hook (source) |
|---|---|---|
| 3 | `/modules` | `useAppModules` — `HomePage.tsx:631`, `entities/app-module/api/queries.ts:24` |
| 4 | `/cycle/today` | `useCycleToday` — `HomePage.tsx:654`, `entities/cycle/api/queries.ts:42` |
| 5 | `/messages/feed?slots=urgent_modal,status_banner,smart_tip,daily_advice,daily_challenge,modal` | `useMessageFeed` — `HomePage.tsx:658`, `entities/message/api/feed.ts:39` |
| 6 | `/profile` | `useUserProfile` — `HomePage.tsx:668`, `entities/user/api/queries.ts:49` |
| 7 | `/fertility/today` | `useFertilityToday` — `HomePage.tsx:681`, `entities/fertility/api/queries.ts:31` |
| 8 | `/home/cycle-overview` | `useCycleOverview` — `HomePage.tsx:745`, `entities/cycle/api/overview.ts:81` |
| 9-10 | `/cycle/month/Y/M?view=calendar` (this month, next month) | `useCycleMonth` ×2 — `HomePage.tsx:754-755`, `entities/cycle/api/queries.ts:123`; `useDayMarks` (`:203-205`) adds up to one more month when the week strip crosses a month |
| 11 | `/banners` | `useBannersSettled` / `BannerSlideshow` ×3 — `HomePage.tsx:881,1422,1467,1475`, `entities/banner/api/queries.ts:23` (one shared request) |
| 12 | `/banners/ads?mode=cycle` | `AdSlot` ×3 — `HomePage.tsx:1424,1474,1476`, `entities/banner/api/queries.ts:68` (one request per mode) |
| 13 | `/home/sections/articles` | `useCycleArticles` — `HomePage.tsx:525`, `entities/article/api/queries.ts:40` |
| 14 | `/auth/user` | `useCurrentUser` via `QuickAccess` — `HomePage.tsx:1312`, `widgets/quick-access/model/use-quick-access.ts:27` |
| 15 | `/todo` | `useTodoBoard` via `ForToday` — `HomePage.tsx:1430`, `entities/todo/api/queries.ts:18` |
| 16 | `/care/today` | `useCareToday` via `TodayRemindersCard` — `HomePage.tsx:1430`, `entities/care-reminder/api/queries.ts:97` |
| 17 | `/checkups/home` | `useCheckupHome` via `CheckupsCard` — `HomePage.tsx:1451`, `entities/checkup/api/queries.ts:52` |
| 18 | `/learning/courses` | `useMyCourses` via `LearnHomeCard` — `HomePage.tsx:1434`, `entities/course/api/queries.ts:27` |
| 19 | `/partners/lab-services` | `useLabServices` via `HomeLabCards` — `HomePage.tsx:1468`, `entities/partner-lab/api/queries.ts:19` |
| 20 | `/home/sections/challenge` | `useTodayChallenge` via `TodayChallengeCard` — `HomePage.tsx:1469`, `entities/challenge/api/queries.ts:25` |
| 21 | `/logs/days/{today}` | `useLogDay` via `TodayLogCard` — `HomePage.tsx:1315`, `entities/health-log/api/log-v2.ts:65` |
| 22 | `/logs/taxonomy` | `useLogTaxonomy` via `TodayLogCard` — `entities/health-log/api/log-v2.ts:29` |
| 23 | `/logs/preferences` | `useLogPreferences` via `TodayLogCard` — `entities/health-log/api/log-v2.ts:47` |
| 24 | `/home/notifications?per_page=50` | `useNotifications` via `HeaderActions` — `HomePage.tsx:160`, `entities/notification/api/queries.ts:24` |
| 25 | `/plus/status` | `usePlusStatus` via `PlusTrialOffer` — `HomePage.tsx:1526`, `entities/plus/api/queries.ts:29` |
| 26 | `/conditions` | `useConditions` via `HomeConditionNudge` — `HomePage.tsx:1432`, `entities/condition/api/queries.ts:24` (module `programs`) |
| 27 | `/messages/nudges` | `useConditionNudges` via `HomeConditionNudge` — `entities/condition/api/queries.ts:69` |
| 28 | `/loss` | `useLossState` via `LossCareReturn` — `HomePage.tsx:1417`, `entities/loss/api/queries.ts:29` |
| 29 | `/companions` | `useCompanions` via `CompanionInviteBanner` — `HomePage.tsx:1421`, `entities/companion/api/queries.ts:35` |
| 30 | `/teen/linked` | `useTeenLinked` via `LinkedTeenCards` — `HomePage.tsx:1419`, `entities/teen/api/parent.ts:129` (module `teen`) |

30 GETs (P-05 estimated 18-20). Not in the replay, on purpose: `/cycle/status` (only while recalculating,
`HomePage.tsx:690`), `/messages/daily` (only when the feed fails, `:662`), `/cycle/date/:d` (only after tapping another
day, `:824`), `/teen/today` (teen mode only), the PWA/plus trial sheet reads.

`web_home.js` checks every response for `200`, so a route that does not exist (or answers 4xx for the seeded cycle user)
shows up as a failed check on the first run.

## 2. Query budgets (`internal/querybudget/budgets.go`)

Counted by the T-M9-07 per-request counter on the real router and MariaDB, auth + module gate included, fixture user of
the route's mode (24 periods, a year of logs). Budget = measured count (the file's convention: a row is lowered to the
measured value, so a regression fails). `teen` and `companion` fixtures are a bare account + life-stage row
(`seedBare` in the int test), pregnancy is the benchfixtures pregnancy user.

| Route | Mode | Cold measured | Warm measured | Budget cold / warm |
|---|---|---:|---:|---|
| `/care/today` | cycle | 5 | 4 | 5 / 4 |
| `/checkups/home` | cycle | 5 | 4 | 5 / 4 |
| `/todo` | cycle | 8 | 6 | 8 / 6 |
| `/home/notifications?per_page=50` | cycle | 4 | 3 | 4 / 3 |
| `/modules` | cycle | 1 | 0 | 1 / 0 |
| `/profile` | cycle | 4 | 3 | 4 / 3 |
| `/fertility/today` | ttc | 6 | 4 | 6 / 4 |
| `/banners/ads?mode=cycle` | cycle | 4 | 3 | 4 / 3 |
| `/learning/courses` | cycle | 5 | 3 | 5 / 3 |
| `/logs/taxonomy` | cycle | 6 | 4 | 6 / 4 |
| `/logs/days/2026-10-20` | cycle | 3 | 2 | 3 / 2 |
| `/pregnancy/v2/today` | pregnancy | 27 | 21 | 27 / 21 |
| `/companion/home` | companion | 9 | 7 | 9 / 7 |
| `/teen/today` | teen | 12 | 7 | 12 / 7 |

Fixtures hold no care/todo/checkup rows, so these are the empty-list costs; growth with rows is the follow-up
check below. Over-budget candidates (follow-ups, not fixed here — outside `touches:`): see the T-M12-30 report
(`/pregnancy/v2/today` 21 warm for a read documented as 5 queries; `/todo` 6; `/teen/today` 7 and
`/companion/home` 7 for near-empty fixtures).

## 3. k6

`loadtest/web_home.js` replays §1 as two `http.batch` waves, thresholds in `loadtest/thresholds.js` (`web_home`):
every request p95 < 60 ms warm / 200 ms cold, the whole two-wave open (`web_home_open`) p95 < 400 ms warm / 1200 ms
cold, errors < 0.1 %. Checked with `k6 inspect` only; **no load run was made against any server** in this task, so
there are no measured k6 results yet (acceptance item left open: run `make loadtest SCENARIO=web_home` after adding
`web_home` to `scenarios=(...)` in `loadtest/run.sh`, which is outside this task's `touches:`).

## T-M12-32 — `user_notifications` listing index (EXPLAIN evidence)

goose 00131 / Laravel twin: `user_notifications_user_id_created_at_id_index (user_id, created_at DESC, id)` — DESC
because the query orders `created_at DESC, id` (mixed directions; an all-ascending index still filesorted on
MariaDB 11.4). 5,000 rows for the user + 5,000 for another:

| | key | extra |
|---|---|---|
| before | `user_notifications_user_id_read_at_index` | Using index condition; Using filesort |
| after (offset 0 / 4980 / 6000) | `user_notifications_user_id_created_at_id_index` | Using where |

Test: `internal/home/store/notifications_index_int_test.go` (drops the index → filesort returns).

## T-M12-31 — `PUT /logs/days/:date` (10-param save)

| | warm | cold |
|---|---:|---:|
| before | 36 | 38 |
| after | 15 | 17 |

Entry writes are now 5 queries for any number of params (user lock, one day read, one `DeleteLogEntryParams`
via FIND_IN_SET, one `InsertLogEntriesBatch` via JSON → digit-table rows, revision bump). The rest is auth, life mode,
custom items, the `daily_health_logs` write-back and the message goal. k6 `log_burst` now drives this route — not run
yet: `make loadtest-db && make loadtest-seed && make loadtest-api && make loadtest SCENARIO=log_burst K6_RUNNER=local`.

## T-M12-34 — Body-hash ETag / 304 on low-churn reads

`etag.ByBody()` on `/modules`, `/banners`, `/banners/ads`, `/logs/taxonomy`, `/health-logs/enums`, `/learning/courses`,
`/checkups/home`: weak tag = SHA-256/2 of the body built for this request; `Vary: Authorization, Accept-Language`;
`Cache-Control: private, no-cache` unless the handler set one. Saves bandwidth, not server work. k6 `web_home` sends
`If-None-Match` from the second open and records `web_home_304` / `web_home_304_bytes_saved` — ratio not measured yet.

## T-M12-27 — `globals.css` split into per-screen CSS (finding P-01)

Measured on a production build (`npm run build`), minified bytes, from the prerendered `/fa/*` HTML
(`frontend/scripts/check-bundle-size.mjs --report`). Lighthouse was **not** run: it needs a logged-in session for
`/fa/home` and the "before" build no longer exists to compare against, so the build numbers are the evidence.

| | before | after |
|---|---:|---:|
| `globals.css` as built (one file, every route) | 812,234 B (793 KB) | 117,145 B (114 KB) |
| CSS every route downloads (globals + CSS reached from the layout) | 793.6 KB | 216.0 KB (221,213 B) |
| `/fa/alerts` (lightest route) | 793.6 KB | 216.0 KB |
| `/fa/calendar` | 793.6 KB | 252.6 KB (31 files) |
| `/fa/log` | 793.6 KB | 251.6 KB (27 files) |
| `/fa/home` (heaviest) | 793.6 KB | 365.1 KB (49 files) |
| median over 158 prerendered routes | 793.6 KB | 228.3 KB |
| `/fa/home` first-load JS (Next table) | 292 kB | 288 kB |
| `/fa/home` first-load JS (script tags in HTML, gzip) | 404.5 KB | 406.2 KB |

What moved: every class family (`nb-` excepted) that is rendered by at most 20 slices became
`<slice>/ui/<prefix>.css` (family used by one slice) or `shared/ui/styles/<prefix>.css` (used by several; imported by every
file that renders it, Next emits it once). A family is detected by the class names present in the CSS
matched against the tokens in the source (exact name, or a `prefix-` stem for dynamic suffixes). Dark-mode
and media-query overrides moved with their rule; selectors that mix a moved and a global family were duplicated per
moved family, so the moved rule still loads after the shared primitive it overrides.

What stays in `globals.css`: tokens, base/reset, Tailwind, shell and shared primitives (`nb-`, `is-`, the
`app-shell`/`home-` bits used by the layout), the families rendered by more than 20 slices, families with no literal
use in source (`cyc-`: built dynamically, left alone), `cal-`/`osheet-`/`pad-`/`fld-`/`jcal-` (see below), keyframes still referenced from global rules.

Cascade finding: a stylesheet imported by a component the root layout reaches is emitted BEFORE `globals.css` in the
HTML, so its rules lose to same-specificity global primitives (the calendar's `.cal-nav-toggle` vs `.chip` padding showed
up in the before/after screenshots). Those five families were folded back into `globals.css`; after that every moved
sheet loads after globals. Rule for new CSS: anything the layout reaches stays global or must win by specificity.

Why "CSS every route downloads" is still 216 KB: the root layout statically imports the sheet registry
(`app/sheets/registry.tsx`) and a few global gates, which pull in the CSS of the lab, plus, companion, record, onboarding
sheets (`lab` 19 KB, `plus` 18 KB, `onb2` 14 KB, `cmp`/`cmh` 42 KB, …). That is JS-structure work, not CSS work: loading
the sheets lazily (`next/dynamic`) would drop that CSS from the global set. Not done here (`app/` is outside this
task's touches) — see the follow-up in the task report.

Gate: `npm run check:bundle` (budgets = measured × 1.1: global 238 KB, heaviest route 402 KB, `/fa/home` JS 447 KB gz).
The style and dark-mode gates now read every `.css` under `src/`.

## T-M12-28 — autopilot, voice-log and mode-specific home branches lazy (findings P-02/P-03)

Production build, `node frontend/scripts/check-bundle-size.mjs --report` (gzip of the `<script src>` tags in the prerendered HTML) and the Next table.

| | before | after |
|---|---:|---:|
| `/fa/home` first-load JS (gzip, gate) | 427,339 B (417.3 KB) | 385,062 B (376.0 KB), -9.9% |
| `/fa/log` first-load JS (gzip, gate) | 407,211 B (397.7 KB) | 372,902 B (364.2 KB), -8.4% |
| `/fa/home` Next table (page / first load) | 57.8 kB / 299 kB | 48.1 kB / 290 kB |
| `/fa/log` Next table | 22.5 kB / 271 kB | 21.2 kB / 259 kB |

What is lazy now:
- Autopilot: `AutopilotMount` keeps one DOM-event listener. `runAutopilot` / `useAutopilotCatalog` call `requestAutopilotRuntime()` (`shared/autopilot/core/wake.ts`), which fires `ritme:autopilot-wake`; the mount then `import()`s the engine + `PILOT_CATALOG` and mounts `AutopilotHost`. `runAutopilot` waits (8 s) for the host and then runs, so callers are unchanged. The catalog test still imports `catalog.ts` statically.
- Voice log: `VoiceLogGated` (recorder, AI-gate wiring, manifest) is `next/dynamic` in `screens/log` (sheet + page), loaded when the voice tab renders (skeleton card meanwhile). `useVoiceLocked` shares a file with the slot, so the screens ask `useAiGate('voice_log')` themselves.
- Home: `MenopauseHome`, `TeenHome` (backdrop fallback, the same markup the page already shows while waiting), `LinkedTeenCards`, `HomeLabCards` (render nothing while loading anyway), `QuickEditSheet`, `PeriodDateEditor`, `SkipPeriodSheet` (mounted on first open).

Verified in the built HTML/chunks: the autopilot catalog/engine (`user_cancelled`, catalog ids), `logs/voice/pilot` + `MediaRecorder`, MenopauseHome, TeenHome, HomeLabCards, LinkedTeenCards and QuickEditSheet strings are absent from the /fa/home and /fa/log first-load scripts.
NOT split yet: `PeriodDateEditor` (chunk 82343) and `SkipPeriodSheet` (in the home page chunk) are still first-load because `HomePage` also imports hooks from the same barrels (`features/log-period`, `features/skip-period`), and webpack keeps every re-exported module of a barrel with side effects. Fix needs those barrels split (outside this task's touches).

Gate: budgets now home 423,936 B and log 410,624 B (measured x 1.1).

## T-M12-33 — Home: ring months fetched in parallel (P-09)

**Change.** `useDayMarks` (HomePage) no longer derives its months from the week strip + ring dates (the ring dates exist only after `/cycle/today`). It now requests previous, current and next Gregorian month at mount through `useCycleMonths` (`useQueries`, same `cycleKeys.month` key + `fetchCycleMonth` as `useCycleMonth`, so cache and in-flight requests are shared). The month set comes from the pure `homeMonthSet` (`entities/cycle/model/homeMonths.ts`); a day outside prev/current/next (long cycle, late period, pregnancy) still adds its own month (fallback), so correctness holds. "Current" is `gregYearMonth(today())` — the same helper the code used before (API months are Gregorian).

**Measurement.** CDP (Playwright + Chrome, fresh context per run, 3 runs each), cold `/fa/home`, production build, +150 ms latency on every request, Go API `CYCLE_ENGINE=v6`, private DB `ritme_t33` (dropped afterwards). Persona: last period 2026-10-20, today 2026-11-21 (cycle day 33, late; ring reaches back into October). Non-preflight API requests in the first 4 s.

| | before | after |
|---|---|---|
| `/cycle/month` requests | 3 (Nov, Dec, Oct) | 3 (Nov, Dec, Oct) |
| all API requests | 32 | 32 |
| October month start (ms after first request) | ~320 (after `/cycle/today` ended at ~316) | ~166 (parallel with `/cycle/today`) |
| critical-path depth to the last month response | 3 | 2 |

Runs: before month-starts 326/320/320 ms, after 166/166/168 ms; today ends ~315 ms in all. Control persona (last period 2026-11-15, ring inside Nov/Dec): before 2 month requests with no waterfall, after 3 (Oct added by prev-month prefetch, still starting at ~165 ms, no extra depth). Note: one more request than before for users whose ring never leaves the current/next month — still ≤ 3.

**Tests.** `homeMonths.test.ts` (month boundary days, year roll-over both ways, leap day, fallback months, malformed/duplicate dates). Mutation check: replacing the `dates` loop with an empty loop fails the fallback test; removing the previous-month entry fails 5 of 7.

## T-M12-29 — Home message payload trimmed with sub-key scopes (finding P-04)

**Change.** A `ROUTE_NAMESPACES` entry may now be a dotted subtree (`menopause.home`, `home.nb.predictions`) next to a whole namespace (`MessageScope`, `shared/i18n/scope.ts`); `pickNamespaces` copies only that subtree (broader entry wins on overlap, missing path skipped, source never mutated). `home` lists the subtrees its components read instead of 23 whole namespaces. Whole namespaces stay where a dynamic key or a handed-over `t` makes the subtree unknowable: `care` (9.5 KB, `doseLine(…, t, …)`), `plus` (11.9 KB, `${base}.sub` in `PlusTrialSheet`), `nav`, `articles`, `challenge`, `profileEdit`, and `logPeriod`/`todo.*`/`companions.recordFor` (cheap to keep coarse). Lazy branches (`MenopauseHome`, `TeenHome`, `LinkedTeenCards`, `HomeLabCards`, `QuickEditSheet`, the trial sheet) render under the same `RouteMessages` provider, so their subtrees (`menopause.home`, `teen.home`, `teen.parent.*`, `homeLab.card`, …) are in the home list; nothing is loaded on mount.

**Measurement** (`npm run build`, `fa`, before = HEAD build, same machine):

| | before | after |
|---|---|---|
| home `RouteMessages` payload, `fa` JSON (`pickNamespaces` on `messages/fa`) | 146,376 B | 56,333 B (-61.5%) |
| same, `en` | 108,143 B | 42,529 B (-60.7%) |
| `.next/server/app/fa/home.html` | 252,443 B | 184,775 B (-67,668 B, -26.8%) |
| `.next/server/app/fa/home.rsc` | 218,576 B | 155,857 B (-62,719 B, -28.7%) |

The HTML/RSC deltas are smaller in percent because they include everything else on the page (the layout's shell namespaces, markup, chunk references), and the messages appear once in the HTML and once more in the inlined RSC flight data.

**Safety net (`message-scopes.test.ts`).** For every route the test walks the import graph (as before) and now also reads every message path each file uses: `const t = useTranslations('a.b')` / `getTranslations(...)` + `t('k')`, `t.rich/raw/has/markup('k')`, template literals (`items.${x}.title` needs the `items` subtree), `t(cond ? 'a' : 'b')` (both), any other expression or a `t` passed to a helper/imported component (the whole prefix subtree). It fails when (1) a needed path that exists in `messages/fa` is outside the route's scope, (2) a dotted entry covers nothing any component reads, (3) a dotted entry does not exist in `messages/fa`, (4) `home` lists a heavy namespace (`home`, `menopause`, `companions`, `teen`, `fertility`, `learn`, `checkups`, `loss`) whole. The same-named `t` of one file is treated as every declaration at once (can only widen, so a need whose path is not in the bundle is skipped).

**A blind spot the work found.** The test's comment stripper removed `/\*…\*/` before line comments, so a line comment containing `` `/fertility/*` `` in `HomePage.tsx` swallowed 15 KB of real code (the first scope draft rendered `home.ring.cycleDay` and `home.nb.lateTextLong` raw on the cycle home). `stripComments` now tokenises strings/templates first; this also corrects the whole-namespace check, which had been reading a truncated `HomePage.tsx` too (it still passes for every route).

**Tests and mutation checks.** `scope.test.ts` (pick: whole, subtree, siblings merge, overlap in both orders, missing path/namespace, source untouched, un-asked subtree absent) and `message-scopes.test.ts` (key finder cases incl. negatives: `format(t)` widens, `obj.t('y')` ignored, `t={t}` to a same-file component does not widen but to an imported one does; `stripComments` cases). Mutations run, each failed the named test, then reverted: drop `menopause.stages` from home (`home: every key its screen reads is inside its scope`); `menopause.home` → `menopause` (heavy-namespace test) and drop `home.title` (coverage); add a typo entry `home.skipp` (dead/misspelled entry); the old naive comment stripper (`stripComments` blind-spot test and `home.days` dead-entry check).

**Manual** (production build, Go API `CYCLE_ENGINE=v6`, private DB `ritme_t29`, contract fixtures, 430 px viewport, light + dark): raw `namespace.key` text (the `e2e/support/raw-keys.ts` detector regex) over `document.body.innerText` plus `MISSING_MESSAGE` console errors: cycle, TTC, menopause and teen homes (`/fa/home`) 0 raw keys / 0 intl errors in light and dark; the pregnancy and postpartum personas redirect to `/fa/pregnancy` and `/fa/postpartum` (their own routes, not changed here) and were clean too. The first draft of the list did show `home.ring.cycleDay` / `home.nb.lateTextLong` on the cycle home — the stripper blind spot above. Screenshots were kept in the session scratchpad, not in the repo.
