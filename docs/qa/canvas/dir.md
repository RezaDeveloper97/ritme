# Mother & child directory — QA (canvas DIR epic)

## CB-DIR-06

Screens rendered with the API mocked at the Playwright network layer (no private DB): one city, 4 categories, 3 places,
one child (14 months), one confirmed booking. 390 wide, 2x.

| Board | Route | Light | Dark | Verdict | Fix |
| --- | --- | --- | --- | --- | --- |
| `nbl_Dir_Home` ([render](canvas/dir/CB-DIR-06/board_Dir_Home.png)) | `/fa/city-services` (top) | [fa](dir/CB-DIR-06/fa_home.light.png) · [en](dir/CB-DIR-06/en_home.light.png) | [fa](dir/CB-DIR-06/fa_home.dark.png) · [en](dir/CB-DIR-06/en_home.dark.png) | Match: location + child header, search with map button, category grid, next booking, age-matched carousel, nearby list, ranking and age notes (with the on/off switch). Gaps: no map preview tile (DIR-07), the board's 8 tiles show only the categories that have a visible place. | n/a |
| `nbl_Dir_Home` (lower half) | `/fa/city-services` (scrolled) | [fa](dir/CB-DIR-06/fa_home_end.light.png) · [en](dir/CB-DIR-06/en_home_end.light.png) | [fa](dir/CB-DIR-06/fa_home_end.dark.png) · [en](dir/CB-DIR-06/en_home_end.dark.png) | Match: nearby rows (open now / verified), join CTA, notes. | Rail cards use ellipsis so long names and distances do not push the card wider. |

Links that point at later tasks: `/city-services/map` (CB-DIR-07) and `/city-services/join` (CB-DIR-10).

## CB-DIR-08

Place page rendered with the API mocked at the Playwright network layer (one verified pool, 3 services, 4 slots incl. a full one, 2 reviews, weekly hours). 390 wide, full height.

| Board | Route | Light | Dark | Verdict | Fix |
| --- | --- | --- | --- | --- | --- |
| `nbl_Dir_Place` ([render](canvas/dir/CB-DIR-08/board_Dir_Place.png)) | `/fa/city-services/[id]` | [fa](dir/CB-DIR-08/fa_place.light.png) · [en](dir/CB-DIR-08/en_place.light.png) | [fa](dir/CB-DIR-08/fa_place.dark.png) · [en](dir/CB-DIR-08/en_place.dark.png) | Match: hero, verified, rating, open state, call/directions/website actions, about + amenity chips, services and prices with «رزرو», nearest slots, health note, reviews with sub-scores, address/weekly hours/rules/cancel, owner link, sticky price + choose-time bar. Gaps: no photo counter, no «message» action (no API), no «report wrong info» link (TODO until the report endpoint lands), reviews list is not a separate «all» page. | Sticky price kept on one line. |
## CB-DIR-07

API mocked at the Playwright network layer (no private DB): one city, 3 categories, 3 places (verified, rating, price-from, next slot / open now), age-range and amenity catalogs. 390 wide, 2x. There is no Neshan key locally, so the map screen can only be shown in its list-only fallback; the map overlay layout (pins, search-this-area, bottom card) comes from the shared `NeshanMap` wrapper (CB-CORE-06) and was not rendered here.

| Board | Route | Light | Dark | Verdict | Fix |
| --- | --- | --- | --- | --- | --- |
| `nbl_Dir_List` ([render](canvas/dir/CB-DIR-07/board_Dir_List.png)) | `/fa/city-services` → category | [fa](dir/CB-DIR-07/fa_list.light.png) · [en](dir/CB-DIR-07/en_list.light.png) | [fa](dir/CB-DIR-07/fa_list.dark.png) · [en](dir/CB-DIR-07/en_list.dark.png) | Match: title with city, search, filter / near-me / open-now chips, result count + sort line, cards with verified badge, rating (count), age fit, price-from and next slot or «open now», floating map button. Gap: the board's «مربی خانم / اتاق شیردهی» chips live in the sheet (amenities) and show as the filter badge count. | n/a |
| `nbl_Dir_List` (filter sheet) | same, «فیلترها» | [fa](dir/CB-DIR-07/fa_filters.light.png) · [en](dir/CB-DIR-07/en_filters.light.png) | [fa](dir/CB-DIR-07/fa_filters.dark.png) · [en](dir/CB-DIR-07/en_filters.dark.png) | Beyond the board (it only shows the chips): child age range, open now switch, amenities, sort. | n/a |
| `nbl_Dir_List` (filtered) | same, after «نمایش نتایج» | [fa](dir/CB-DIR-07/fa_list_filtered.light.png) · [en](dir/CB-DIR-07/en_list_filtered.light.png) | [fa](dir/CB-DIR-07/fa_list_filtered.dark.png) · [en](dir/CB-DIR-07/en_list_filtered.dark.png) | Match: active filters badge and the «open now» chip on. | n/a |
| `nbl_Dir_Map` ([render](canvas/dir/CB-DIR-07/board_Dir_Map.png)) | `/fa/city-services/map` (no key → list fallback) | [fa](dir/CB-DIR-07/fa_map_nokey.light.png) · [en](dir/CB-DIR-07/en_map_nokey.light.png) | [fa](dir/CB-DIR-07/fa_map_nokey.dark.png) · [en](dir/CB-DIR-07/en_map_nokey.dark.png) | Fallback match: search, filter button with count, category chips, then the same cards. With a key: pins, «search this area» (bbox), selected-place card and «list · N places» toggle. | n/a |

The dev-server «N» badge in the corner of the screenshots is Next's dev indicator, not part of the app.
## CB-DIR-10

Application API mocked at the Playwright network layer (stateful mock: save, upload, 422 then success). 390 wide, 2x, tall viewport so each step is one image.

| Board | Route | Light | Dark | Verdict | Fix |
| --- | --- | --- | --- | --- | --- |
| `nbl_Dir_Join` ([render](canvas/dir/CB-DIR-10/board_Dir_Join.png)) | `/fa/city-services/join (step 1)` | [fa](dir/CB-DIR-10/fa_1_intro.light.png) · [en](dir/CB-DIR-10/en_1_intro.light.png) | [fa](dir/CB-DIR-10/fa_1_intro.dark.png) · [en](dir/CB-DIR-10/en_1_intro.dark.png) | Match: stepper, benefits, cooperation-terms placeholder (no invented commission), name, multi-select types, city, district, contact, owner confirm. Delete-draft added. | n/a |
| `nbl_Dir_JoinForm` ([render](canvas/dir/CB-DIR-10/board_Dir_JoinForm.png)) | `step 2` | [fa](dir/CB-DIR-10/fa_2_place.light.png) · [en](dir/CB-DIR-10/en_2_place.light.png) | [fa](dir/CB-DIR-10/fa_2_place.dark.png) · [en](dir/CB-DIR-10/en_2_place.dark.png) | Match: pin, address, photos grid (min 4), about, ages, amenities, weekly hours. Gap: Neshan map shows only with NEXT_PUBLIC_NESHAN_KEY; without it the device-location button and typed coordinates are shown (shot is the no-key state). | Textarea and coordinate fields sized for the long text. |
| `nbl_Dir_JoinDocs (top)` ([render](canvas/dir/CB-DIR-10/board_Dir_JoinDocs (top).png)) | `step 3` | [fa](dir/CB-DIR-10/fa_3_services.light.png) · [en](dir/CB-DIR-10/en_3_services.light.png) | [fa](dir/CB-DIR-10/fa_3_services.dark.png) · [en](dir/CB-DIR-10/en_3_services.dark.png) | Match: service rows with price, add/edit in a sheet (also `3_sheet_after`, `3_services_empty`). | n/a |
| `nbl_Dir_JoinDocs` ([render](canvas/dir/CB-DIR-10/board_Dir_JoinDocs.png)) | `step 4` | [fa](dir/CB-DIR-10/fa_4_docs.light.png) · [en](dir/CB-DIR-10/en_4_docs.light.png) | [fa](dir/CB-DIR-10/fa_4_docs.dark.png) · [en](dir/CB-DIR-10/en_4_docs.dark.png) | Match: booking mode, private documents (licence required, health cert for pools), terms. Booking copy says requests are confirmed by the Ritme team, not auto-confirmed. | n/a |
| `nbl_Dir_JoinDocs (422)` ([render](canvas/dir/CB-DIR-10/board_Dir_JoinDocs (422).png)) | `step 4 submit error` | [fa](dir/CB-DIR-10/fa_4_docs_missing.light.png) · [en](dir/CB-DIR-10/en_4_docs_missing.light.png) | [fa](dir/CB-DIR-10/fa_4_docs_missing.dark.png) · [en](dir/CB-DIR-10/en_4_docs_missing.dark.png) | Per-field `application_incomplete` errors with a link back to the owning step. | n/a |
| `nbl_Dir_JoinDone` ([render](canvas/dir/CB-DIR-10/board_Dir_JoinDone.png)) | `done` | [fa](dir/CB-DIR-10/fa_5_done.light.png) · [en](dir/CB-DIR-10/en_5_done.light.png) | [fa](dir/CB-DIR-10/fa_5_done.dark.png) · [en](dir/CB-DIR-10/en_5_done.dark.png) | Match: tracking code, timeline (submitted, documents review, expert contact, published), add-more-photos, support link. Review time left out (TBD, not invented). | n/a |

## CB-DIR-09

API mocked at the Playwright network layer (no private DB): one online pool with 2 services, 7 slots (one full), one own child, upcoming (requested + confirmed) and past (completed + cancelled) bookings. 390 wide, 2x, full page; the dev-server «N» badge is Next's, not the app.

| Board | Route | Light | Dark | Verdict | Fix |
| --- | --- | --- | --- | --- | --- |
| `nbl_Dir_Book` ([render](canvas/dir/CB-DIR-09/board_Dir_Book.png)) | `/fa/city-services/[id]/request?service=&slot=` | [fa book](dir/CB-DIR-09/fa_book.light.png) · [en book](dir/CB-DIR-09/en_book.light.png) | [fa book](dir/CB-DIR-09/fa_book.dark.png) · [en book](dir/CB-DIR-09/en_book.dark.png) | Match: services as radio cards, own-child chips (first name + age, «add child»), day strip, time grid with full / seats left, note, summary, data-sharing note, sticky bar. Board copy «پرداخت و رزرو / پرداخت شد» became «ارسال درخواست» + «در محل پرداخت می‌شود» (no payment, DECISIONS); a request note says it is not booked until the team confirms. Gap: the day strip lists only days that have slots (no greyed empty days); the summary shows the venue's free-cancel hours. | Fix: day/slot params read by the server page. |
| `nbl_Dir_Booked` ([render](canvas/dir/CB-DIR-09/board_Dir_Booked.png)) | `/fa/city-services/booked/[id]` | [fa booked](dir/CB-DIR-09/fa_booked.light.png) · [en booked](dir/CB-DIR-09/en_booked.light.png) | [fa booked](dir/CB-DIR-09/fa_booked.dark.png) · [en booked](dir/CB-DIR-09/en_booked.dark.png) | Match: status header, service/venue, day + time, child + age, price line, directions and call, cancel, cancel window, all-bookings link. Honest status: «درخواستت ثبت شد / در انتظار تأیید» until the API says confirmed. Gaps: no «what to bring» list and no «added to the Ritme calendar» toggle (neither exists in the API; nothing claimed). | n/a |
| `nbl_Dir_MyBookings` ([render](canvas/dir/CB-DIR-09/board_Dir_MyBookings.png)) | `/fa/profile/bookings` | [fa mybookings](dir/CB-DIR-09/fa_mybookings.light.png) · [en mybookings](dir/CB-DIR-09/en_mybookings.light.png) · [fa past](dir/CB-DIR-09/fa_mybookings_past.light.png) · [en past](dir/CB-DIR-09/en_mybookings_past.light.png) | [fa mybookings](dir/CB-DIR-09/fa_mybookings.dark.png) · [en mybookings](dir/CB-DIR-09/en_mybookings.dark.png) · [fa past](dir/CB-DIR-09/fa_mybookings_past.dark.png) · [en past](dir/CB-DIR-09/en_mybookings_past.dark.png) | Match: upcoming/past tabs with counts, rows with status pill, directions, change time, cancel, past: review (pending until moderated) and book again; doctor appointments stay linked. Gaps: no «message the venue» (no API), no multi-session package balance (not in the API yet), the thumbnail is the category icon when the venue has no photo. | n/a |
| Report wrong info (PlaceInfo TODO from CB-DIR-08) | `/fa/city-services/[id]` | [fa](dir/CB-DIR-09/fa_report.light.png) · [en](dir/CB-DIR-09/en_report.light.png) | [fa](dir/CB-DIR-09/fa_report.dark.png) · [en](dir/CB-DIR-09/en_report.dark.png) | Beyond the board: reason chips + optional note, sent to POST /listings/{id}/report; worked by the team, never the venue. | n/a |

## CB-DIR-11

Desktop web, 1440 wide, logged-out except Booked (fake session). Board renders were not re-shot for this task (compared against `docs/design/canvas-v1/text/w.txt`). The Neshan map cannot render without `NEXT_PUBLIC_NESHAN_KEY`, so the screenshots show the documented list fallback (map column hidden). Lighthouse SEO on `/fa/places/1` (production build, mock API): 100.

| Board | Route | Light | Dark | Verdict | Fix |
| --- | --- | --- | --- | --- | --- |
| `W_Dir_Search` | `/fa/places` | [search](dir/CB-DIR-11/search-light.png) | [search](dir/CB-DIR-11/search-dark.png) | Match: header nav + login, hero, search + city, category chips, filter sidebar (age, open now, amenities, sort), result cards, pager, rank note; sticky map column (shared NeshanMap) when a key exists. Gaps: no child-age field in the hero (age is a sidebar filter), no «search as the map moves» toggle (the map's «search this area» button is used). | n/a |
| `W_Dir_Place` | `/fa/places/[id]` | [place](dir/CB-DIR-11/place-light.png) · [logged-out booking](dir/CB-DIR-11/place-login-light.png) | [place](dir/CB-DIR-11/place-dark.png) | Match: breadcrumb, verified badge, rating, call/directions/website, gallery, about + amenities, services & prices, weekly hours, reviews, address/rules/cancel, sticky booking panel (service, slots, login link when logged out). Gaps: no share/save buttons, no photo lightbox. | Fix: key per photo; actions row. |
| `W_Dir_Booked` | `/fa/places/booked/[id]` | [booked](dir/CB-DIR-11/booked-light.png) | [booked](dir/CB-DIR-11/booked-dark.png) | Match: honest status hero (request until confirmed), rows, directions, call, app hint. Gaps: no store badges (no real links), no «what to bring» (no API), cancel/reschedule stay in the app (link). | n/a |

## CB-DIR-12

Desktop web, 1440 wide (landing and login prompt 1x, wizard and done 2x), API mocked at the Playwright network layer (fake session for the wizard; landing and login prompt logged out). Board renders were not re-shot (compared against the board text). The wizard reuses the app's step forms through the new `widgets/city-join-form` (moved out of `screens/city-services-join`), so it is the same draft API and a draft continues in the app. The cooperation terms are a neutral placeholder («model not set yet, the team shares terms before publishing»): no admin-editable group exists for it (`info_sections` groups are a fixed backend list) and no fee is stated.

| Board | Route | Light | Dark | Verdict | Fix |
| --- | --- | --- | --- | --- | --- |
| `W_Dir_Business` | `/fa/business` | [fa](dir/CB-DIR-12/fa_landing.light.png) · [en](dir/CB-DIR-12/en_landing.light.png) | [fa](dir/CB-DIR-12/fa_landing.dark.png) · [en](dir/CB-DIR-12/en_landing.dark.png) | Match: hero + CTA, why (3), 4 steps, requirements, cooperation terms, FAQ (accordion + FAQPage JSON-LD), closing CTA, SSR metadata. Gaps: the board's sample venue card is a neutral benefits panel (no invented venue); review time is not stated (team announces it). | Fix: H1 size. |
| `W_Dir_Join` | `/fa/business/join` | [fa step 1](dir/CB-DIR-12/fa_1_intro.light.png) · [fa step 2](dir/CB-DIR-12/fa_2_place.light.png) · [fa step 4](dir/CB-DIR-12/fa_4_docs.light.png) · [en step 1](dir/CB-DIR-12/en_1_intro.light.png) · [login](dir/CB-DIR-12/fa_login.light.png) | [fa step 1](dir/CB-DIR-12/fa_1_intro.dark.png) · [fa step 2](dir/CB-DIR-12/fa_2_place.dark.png) · [fa step 4](dir/CB-DIR-12/fa_4_docs.dark.png) · [en step 1](dir/CB-DIR-12/en_1_intro.dark.png) · [login](dir/CB-DIR-12/fa_login.dark.png) | Match: step rail (done/current), save and exit, autosave hint, draft card preview, form column, previous step. Gaps: step forms are the app's single-column forms (no two-column desktop form). | n/a |
| `W_Dir_JoinDone` | `/fa/business/join` (after submit) | [fa](dir/CB-DIR-12/fa_5_done.light.png) · [en](dir/CB-DIR-12/en_5_done.light.png) | [fa](dir/CB-DIR-12/fa_5_done.dark.png) · [en](dir/CB-DIR-12/en_5_done.dark.png) | Match: confirmation, tracking code + copy, review timeline, support link. Gaps: no «preview venue page» / «complete free slots» actions (page not published yet). | n/a |
