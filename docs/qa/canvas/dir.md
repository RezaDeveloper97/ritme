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
