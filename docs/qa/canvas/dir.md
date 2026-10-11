# Mother & child directory — QA (canvas DIR epic)

## CB-DIR-06

Screens rendered with the API mocked at the Playwright network layer (no private DB): one city, 4 categories, 3 places,
one child (14 months), one confirmed booking. 390 wide, 2x.

| Board | Route | Light | Dark | Verdict | Fix |
| --- | --- | --- | --- | --- | --- |
| `nbl_Dir_Home` ([render](canvas/dir/CB-DIR-06/board_Dir_Home.png)) | `/fa/city-services` (top) | [fa](dir/CB-DIR-06/fa_home.light.png) · [en](dir/CB-DIR-06/en_home.light.png) | [fa](dir/CB-DIR-06/fa_home.dark.png) · [en](dir/CB-DIR-06/en_home.dark.png) | Match: location + child header, search with map button, category grid, next booking, age-matched carousel, nearby list, ranking and age notes (with the on/off switch). Gaps: no map preview tile (DIR-07), the board's 8 tiles show only the categories that have a visible place. | n/a |
| `nbl_Dir_Home` (lower half) | `/fa/city-services` (scrolled) | [fa](dir/CB-DIR-06/fa_home_end.light.png) · [en](dir/CB-DIR-06/en_home_end.light.png) | [fa](dir/CB-DIR-06/fa_home_end.dark.png) · [en](dir/CB-DIR-06/en_home_end.dark.png) | Match: nearby rows (open now / verified), join CTA, notes. | Rail cards use ellipsis so long names and distances do not push the card wider. |

Links that point at later tasks: `/city-services/map` (CB-DIR-07) and `/city-services/join` (CB-DIR-10).
