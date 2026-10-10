# NAV — design fidelity (canvas-build §5)

Board renders: `docs/qa/canvas/boards/<board>.png` (both NAV boards are dark-only; light is built from the current
tokens). Screens: 390 px, fa, headless Chrome over CDP against the local Go API :8020 + Next dev :3000, test user
`09120000427` (cycle mode; three abdominal-pain days logged 2026-09-29…10-01 so «از ثبت‌های تو» has a hit). Colours are
checked against the token map (`docs/canvas-build/README.md` §4), not the board hex.

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbd_Nav_Search` | `/search` | [recents](nav/CB-NAV-02/search-recents-light.png) · [«درد»](nav/CB-NAV-02/search-dard-light.png) · [«قرص»](nav/CB-NAV-02/search-ghors-light.png) · [no results](nav/CB-NAV-02/search-empty-light.png) | [recents](nav/CB-NAV-02/search-recents-dark.png) · [«درد»](nav/CB-NAV-02/search-dard-dark.png) · [«قرص»](nav/CB-NAV-02/search-ghors-dark.png) · [no results](nav/CB-NAV-02/search-empty-dark.png) | ✔ | CB-NAV-02. Same order and hierarchy: 52 px pill `SearchField` (focus = `--brand` border + ring) with «لغو» at the end (→ `/home`), horizontally scrolling scope chips (selected = `--brand-soft` + `--brand` border), grouped flat cards with icon discs + chevrons, «همه N» end link on a group when it has more than shown (switches the scope), shop note card, «جست‌وجوهای اخیر» + «پاک کردن» + clock chips. No bottom nav (flow). Deliberate differences: chips are the API scopes — «خدمات» instead of the board's «پزشک» (doctors/consultation don't exist yet; services = checkups + reminders) and the services group is titled «خدمات و مراقبت»; insight row reads «۳ روز با درد در ۳۰ روز اخیر» when the window is 30 days (board's «در این چرخه» shows when the API window is the cycle); «بیشترین شدت: ۶ از ۱۰» drops «روز X» when the API has no cycle day; «الگوی درد» + «در تحلیل» instead of «الگوی درد در ۶ چرخه اخیر» (the API doesn't return a cycle count). Hits whose screen isn't built (`/programs/pelvic`) are hidden; articles open the article sheet. Extra states with no board: start (no recents), «دست‌کم ۲ حرف بنویس», loading skeleton, error + retry, no results. |
| `nbd_Nav_Today` (header search) | `/home` | [light](nav/CB-NAV-02/home-header-light.png) | [dark](nav/CB-NAV-02/home-header-dark.png) | ✔ | CB-NAV-02 scope only: 44 px round search button before the bell in the Today header (cycle / TTC / teen / postpartum via `HomeHeader`, menopause via the same header, pregnancy `/pregnancy` hub via `HeaderButton`). Pregnancy header not screenshotted (test user is in cycle mode); covered by typecheck + message-scope test. Mode chip, «برای امروز», quick access = CB-NAV-03. |

## CB-NAV-03

IA alignment of bloom's shell with the canvas-v1 IA rules (IA_Nav, IA_Map). Screens: 390 px, fa, Go API :8291 on a
copy of ritme_dev (`ritme_nav03`, migrated to goose 54; ritme_dev untouched) + Next dev :3122. Test users: cycle
`09900000004` (child «آوا» inserted in the copy for the + sheet row), teen `09120007702`, menopause `09120005055`,
IVF `09900002021`. Shots in `nav/CB-NAV-03/`.

### Rule check (IA_Nav / IA_Map)

| Rule | Before | Fix (test / QA) |
|---|---|---|
| Tab bar hidden on forms / log / checkout / loss / lock | `/log` and `/pregnancy/calendar` still in the B-N1-04 «transitional» list (both have back/close headers now); `/loss*`, checkout, forms, lock already hidden | removed both from `NAV_ROOT_PATHS`; test `CB-NAV-03 IA_Nav rules › hides the nav…` |
| + always opens the mode's log sheet | ✔ FAB → `?sheet=log` in every owner mode (companion: no FAB) | test `the FAB is present in every owner mode` |
| Re-tap active tab → tab root | from a sub-screen the tab link already went to the root; on the root itself a re-tap did nothing | `isRootRetap` + scroll the screen back to top (reduced motion → instant); test `re-tap on the active tab root…` |
| Stage tab label/icon per mode, fixed position | ✔ (IVF «درمان» CB-IVF-02, menopause «علائم» CB-MENO-08) | test `keeps the stage tab in the second slot…` |
| Stage tab (cycle): تقویم · تحلیل · تاریخچه | segments ماه · سال · تحلیل; history `/cycle` reachable only from the home predictions card | 4th segment «تاریخچه» → `/cycle` (tab stays lit) — [light](nav/CB-NAV-03/fa_calendar.light.png) · [dark](nav/CB-NAV-03/fa_calendar.dark.png) |
| Services order; shop last in its own frame + note; 115 card | ✔ with the seeded catalog, but the order was fully admin-editable | `iaOrderedSections`: shop always last content section, emergency at the end; test `ia-order.test.ts` — [light](nav/CB-NAV-03/fa_services.light.png) · [dark](nav/CB-NAV-03/fa_services.dark.png) |
| Teen Services: no shop, no ads; doctor & labs only with a parent | no shop / banners ✔ (CB-TEEN-01); doctor + lab tiles shown to every teen | backend `services.WithTeenParent`: teen without an active parent link gets no `doctors` / `labs` tile; Go test `TestHub_TeenDoctorAndLabsOnlyWithParent` — [light](nav/CB-NAV-03/fa_services.teen.light.png) · [dark](nav/CB-NAV-03/fa_services.teen.dark.png) |
| Me: no health tool reachable only from Me | analysis (stage tab), vitals / record (Services), cycle settings (Today cog) ✔; **to-do list only from Me** | to-do count row in «برای امروز» → `/todo` |
| Me: bookings + orders rows | ✔ «به‌زودی» rows (wiring = CB-DIR-09 / CB-SHOP-08); «فرزندان» was «به‌زودی» though `/children` shipped (B-N5-05) | children row live → `/children` (teen: hidden) — [light](nav/CB-NAV-03/fa_profile.light.png) · [dark](nav/CB-NAV-03/fa_profile.dark.png) |
| Today header: mode chip → mode screen + search | search ✔ except the teen home (CB-TEEN-02 header had none); no mode chip | `widgets/mode-chip` on cycle/TTC/menopause/teen (`HomeHeader`, teen slot), pregnancy + IVF (`HubHeader` date), postpartum (replaces the static mode label); search added to the teen header; test `chip.test.ts` |
| Today «برای امروز» = reminders + to-dos + companion suggestion | M3 «یادآورهای امروز» card alone | `widgets/for-today` wraps the reminders card + «N کار امروز» row (hidden at 0 / without the API); companion row hidden (no API) → CB-NAV-03b |
| Today «دسترسی سریع» (customisable) | none | `widgets/quick-access` + edit sheet `?sheet=quick-access` (≤ 9, on-device per README §8 Q3, account-stamped, wiped at sign-out); only shipped screens are offered; tests `quick-access.test.ts` |
| + sheet «برای بعد» + «ثبت برای <child>» | none (postpartum preset had baby rows) | `widgets/log-later` via a new `extra` slot of `LogDay`: appointment → `/reminders/appointment/new`, task → `?sheet=todo-add`, child row → `/children/<id>/feeding` (not in postpartum) |

### Fidelity

| Board | Route | Light | Dark | Verdict | Fix / note |
|---|---|---|---|---|---|
| `nbd_Nav_Today` | `/home` (cycle) | [light](nav/CB-NAV-03/fa_home.light.png) | [dark](nav/CB-NAV-03/fa_home.dark.png) | ✔ | Chip above bloom's date/greeting (board: same row as the buttons, top-start), «برای امروز» (reminders card + to-do row), «دسترسی سریع» 4 defaults + «ویرایش». Board's doctor / pain diary / courses tiles are hidden until their screens exist (defaults fall back to checkups, reminders, analysis). Ring/hero = bloom. |
| `nbd_Nav_Today` (other modes) | `/home` teen · menopause, `/ivf` | [teen](nav/CB-NAV-03/fa_home.teen.light.png) · [meno](nav/CB-NAV-03/fa_home.meno.light.png) · [ivf](nav/CB-NAV-03/fa_ivf.ivf.light.png) | [teen](nav/CB-NAV-03/fa_home.teen.dark.png) · [meno](nav/CB-NAV-03/fa_home.meno.dark.png) · [ivf](nav/CB-NAV-03/fa_ivf.ivf.dark.png) | ✔ | Chip per mode tone (teen brand, menopause data, IVF warm «IVF / IUI»). Menopause quick access sits under the menopause content (hot flash first). Pregnancy / postpartum not screenshotted (no test user in those modes on the copy) — typecheck + scope test. |
| — (edit sheet, no board) | `?sheet=quick-access` | [light](nav/CB-NAV-03/fa_home_sheet_quick-access.light.png) | [dark](nav/CB-NAV-03/fa_home_sheet_quick-access.dark.png) | ✔ | Minimal (DECISIONS #8): count «۴ از ۹», icon + checkbox rows, «ذخیره», «برگشت به پیش‌فرض». |
| `nbd_Nav_Plus` | `?sheet=log` | [light](nav/CB-NAV-03/fa_home_sheet_log.light.png) | [dark](nav/CB-NAV-03/fa_home_sheet_log.dark.png) | ✔ | «برای بعد» pair + «ثبت برای آوا» card at the end of the manual tab (board: under the tiles — CB-NAV-03b). Bloom's tiles/accordion unchanged. |
| `nbd_Nav_Services` | `/services` | [light](nav/CB-NAV-03/fa_services.light.png) | [dark](nav/CB-NAV-03/fa_services.dark.png) | ✔ | Order and dashed shop frame match; program tiles' targets belong to CB-COND-02. |
| `nbd_Nav_Stage` | `/calendar` | [light](nav/CB-NAV-03/fa_calendar.light.png) | [dark](nav/CB-NAV-03/fa_calendar.dark.png) | ✔ | Segments ماه · سال · تحلیل · تاریخچه (board: تقویم · تحلیل · تاریخچه; bloom keeps its month/year split). |
| `nbd_Nav_Me` | `/profile` | [light](nav/CB-NAV-03/fa_profile.light.png) | [dark](nav/CB-NAV-03/fa_profile.dark.png) | ✔ | Bloom B-N1-10 groups; «فرزندان» live. |
| `nbd_Nav_Mode` | `/profile/mode` | — | — | ✔ | Bloom B-N2-03 screen unchanged (now also reached from the chip). |

## CB-NAV-03b

IA follow-ups of CB-NAV-03 (boards `nbd_Nav_Today`, `nbd_Nav_Plus`). No new CSS (the CB-NAV-03 block styles the
reused widgets); the dead `.pp-brand-mode` rule was removed.

| Item | Status | Change / note |
|---|---|---|
| «برای امروز» + «دسترسی سریع» on the postpartum home | ✔ | `/postpartum`: `ForToday` (M3 reminders + to-do row) + `QuickAccess` after the tiles, before «ویزیت‌ها» (defaults: children, record, checkups, reminders); scopes `postpartum` / `postpartumSetup` + `care`, `todo` |
| … on the IVF home | ✔ | `/ivf`: both after the IVF quick actions (defaults: ivfMeds, record, labs, reminders); scope `ivf` + `care`, `todo` |
| … on the pregnancy home | ✔ | had quick access only; the reminders card is now wrapped in `ForToday` (adds the to-do row); scope `pregnancy` + `todo` |
| + sheet «برای بعد» position | ✔ | `LogDay` `extra` slot moved from after the full category list to right under the quick tiles / mode preset (board order) |
| Dead key `postpartum.home.modeLabel` | ✔ | removed from frontend fa/en, Go copy fa/en and i18n goldens ar/en/fa; `go test ./internal/i18n` green |
| Shortcut targets «پزشک» / «دفترچه درد» / «دوره‌ها» | open | still no screens: no doctor directory route, `/programs` is only the CB-COND-02 hub (no pain diary), `/learn` is only registered in the nav list (B-N8-03 page not shipped) — `href` stays `null` |
| Companion suggestion row in «برای امروز» | open | no owner-facing companion-suggestion API in bloom N4 |
| Quick-access server persistence (README §8 Q3) | open | awaiting the user's choice; stays on-device |

Verify: tsc, eslint, steiger, lint:styles, lint:dark, vitest (widgets/app/postpartum/ivf/pregnancy/log-day: 385),
`go test ./internal/i18n`. Screenshots (`nav/CB-NAV-03b/`) not taken: the docker test-stack MariaDB stopped
responding mid-run (API ping timeout on the `ritme_nav03b` copy), so postpartum (`09900005501`) / IVF
(`09900002021`) shots are pending a re-run.
