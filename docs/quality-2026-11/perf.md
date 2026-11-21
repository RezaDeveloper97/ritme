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
