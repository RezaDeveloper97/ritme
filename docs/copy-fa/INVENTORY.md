# M8 namespace inventory

All 70 namespaces of `frontend/messages/fa/` (2026-10-18, before M8). **Colloq / written** = rough regex count of
colloquial markers (می‌کنه، میشه، یه، رو، اومد، اگه، توی …) vs written markers (را، می‌شود، است، برایت، اگر …).
Target register for all: written-but-warm (STYLE.md §1). Checker baseline counts are appended by T-M8-02.

| Namespace | Bytes | Strings | Colloq / written | Register now | Sensitivity | Batch |
|---|---|---|---|---|---|---|
| `account` | 816 | 7 | 10 / 0 | colloquial | legal | T-M8-03 |
| `analysis-postpartum` | 4,063 | 68 | 3 / 8 | written | medical | T-M8-07 |
| `analysis-pregnancy` | 6,670 | 94 | 0 / 14 | written | medical | T-M8-06 |
| `analysis` | 31,551 | 431 | 1 / 78 | written | medical | T-M8-04 |
| `articles` | 1,879 | 32 | 7 / 0 | colloquial | UI | T-M8-13 |
| `assistant` | 11,066 | 160 | 3 / 29 | written | UI | T-M8-13 |
| `auth` | 1,820 | 26 | 4 / 7 | mixed | legal | T-M8-03 |
| `baby-log` | 4,618 | 90 | 0 / 12 | written | UI | T-M8-07 |
| `banners` | 165 | 4 | 0 / 0 | — | UI | T-M8-03 |
| `calendar` | 5,033 | 99 | 13 / 7 | mixed | medical | T-M8-04 |
| `care` | 10,289 | 218 | 6 / 11 | mixed | medical | T-M8-09 |
| `challenge` | 587 | 10 | 0 / 1 | written | UI | T-M8-03 |
| `checkups` | 10,265 | 187 | 0 / 35 | written | medical | T-M8-09 |
| `children` | 12,822 | 237 | 0 / 34 | written | UI | T-M8-07 |
| `city-join` | 13,218 | 215 | 0 / 47 | written | commercial | T-M8-12 |
| `city-services` | 14,145 | 252 | 2 / 53 | written | UI | T-M8-12 |
| `common` | 3,685 | 62 | 0 / 13 | written | UI | T-M8-03 |
| `companion-home` | 5,821 | 97 | 2 / 17 | written | UI | T-M8-07 |
| `companions` | 11,848 | 174 | 1 / 55 | written | UI | T-M8-07 |
| `conditions` | 9,512 | 145 | 1 / 27 | written | medical | T-M8-09 |
| `contraception` | 7,979 | 137 | 0 / 18 | written | medical | T-M8-10 |
| `cycle` | 5,871 | 98 | 6 / 10 | mixed | medical | T-M8-04 |
| `day-tasks` | 700 | 13 | 2 / 0 | colloquial | UI | T-M8-03 |
| `doctors` | 8,995 | 157 | 0 / 20 | written | UI | T-M8-12 |
| `fertility` | 9,691 | 153 | 11 / 6 | mixed | medical | T-M8-06 |
| `health-record` | 7,212 | 138 | 0 / 4 | written | medical | T-M8-08 |
| `home-lab` | 5,049 | 75 | 1 / 19 | written | medical | T-M8-08 |
| `home` | 13,734 | 247 | 29 / 13 | colloquial | medical | T-M8-04 |
| `insurance` | 22,730 | 360 | 1 / 69 | written | commercial | T-M8-11 |
| `ivf` | 21,657 | 355 | 1 / 66 | written | medical, emotional | T-M8-06 |
| `lab-consent` | 1,039 | 14 | 0 / 6 | written | legal | T-M8-03 |
| `labs` | 10,377 | 177 | 1 / 29 | written | medical | T-M8-08 |
| `learn` | 7,822 | 139 | 0 / 23 | written | UI | T-M8-13 |
| `log-customize` | 3,527 | 61 | 1 / 4 | written | UI | T-M8-05 |
| `log-period` | 2,527 | 49 | 6 / 4 | mixed | UI | T-M8-05 |
| `log-sheet` | 9,338 | 177 | 3 / 16 | written | UI | T-M8-05 |
| `log-taxonomy` | 15,549 | 283 | 0 / 0 | — | medical | T-M8-05 |
| `log` | 7,789 | 174 | 6 / 0 | colloquial | UI | T-M8-05 |
| `loss` | 7,891 | 121 | 3 / 33 | written | emotional | T-M8-06 |
| `me` | 25,876 | 420 | 1 / 56 | written | UI | T-M8-13 |
| `menopause` | 19,580 | 342 | 1 / 37 | written | medical | T-M8-10 |
| `nav` | 3,117 | 75 | 2 / 1 | mixed | UI | T-M8-03 |
| `notifications` | 371 | 8 | 2 / 0 | colloquial | UI | T-M8-03 |
| `onboarding` | 10,963 | 167 | 3 / 45 | written | legal | T-M8-03 |
| `pelvic` | 3,803 | 63 | 0 / 9 | written | medical | T-M8-07 |
| `phase-details` | 1,833 | 37 | 1 / 5 | written | medical | T-M8-04 |
| `plus` | 10,480 | 168 | 1 / 30 | written | commercial | T-M8-11 |
| `postpartum` | 8,200 | 147 | 2 / 26 | written | medical | T-M8-07 |
| `pregnancy-tools` | 5,702 | 76 | 0 / 30 | written | medical | T-M8-06 |
| `pregnancy-v2` | 17,226 | 281 | 49 / 2 | colloquial | medical | T-M8-06 |
| `pregnancy` | 9,037 | 161 | 31 / 1 | colloquial | medical | T-M8-06 |
| `profile-edit` | 1,447 | 34 | 5 / 0 | colloquial | UI | T-M8-03 |
| `profile-info` | 6,630 | 4 | 0 / 0 | — | UI | T-M8-03 |
| `profile` | 2,269 | 45 | 2 / 2 | mixed | UI | T-M8-03 |
| `pwa` | 1,354 | 15 | 14 / 0 | colloquial | UI | T-M8-03 |
| `record-export` | 8,121 | 133 | 0 / 24 | written | UI | T-M8-08 |
| `record` | 23,133 | 396 | 2 / 70 | written | medical | T-M8-08 |
| `reminders` | 2,006 | 41 | 5 / 0 | colloquial | UI | T-M8-09 |
| `search` | 2,360 | 43 | 0 / 3 | written | UI | T-M8-03 |
| `services` | 2,224 | 37 | 1 / 4 | written | UI | T-M8-12 |
| `shop` | 21,074 | 374 | 1 / 58 | written | commercial | T-M8-11 |
| `teen` | 8,112 | 116 | 0 / 37 | written | emotional | T-M8-10 |
| `todo` | 4,139 | 82 | 0 / 7 | written | UI | T-M8-13 |
| `vitals` | 10,569 | 209 | 0 / 18 | written | medical | T-M8-09 |
| `voice-log` | 4,896 | 71 | 1 / 21 | written | UI | T-M8-05 |
| `web-business` | 6,722 | 69 | 0 / 28 | written | commercial | T-M8-12 |
| `web-directory` | 2,928 | 38 | 1 / 8 | written | UI | T-M8-12 |
| `web-shell` | 689 | 13 | 0 / 1 | written | UI | T-M8-03 |
| `web-shop` | 4,382 | 50 | 0 / 9 | written | commercial | T-M8-11 |
| `welcome` | 3,117 | 45 | 1 / 5 | written | UI | T-M8-03 |

## Batches

| Batch | Namespaces | Rewriter | Notes |
|---|---|---|---|
| T-M8-03 | `common`, `nav`, `auth`, `welcome`, `onboarding`, `account`, `profile`, `profile-edit`, `profile-info`, `pwa`, `web-shell`, `notifications`, `banners`, `search`, `day-tasks`, `challenge`, `lab-consent` | sonnet + opus review | pilot, runs alone |
| T-M8-04 | `home`, `cycle`, `calendar`, `phase-details`, `analysis` | sonnet + opus review | parallel group M8-C |
| T-M8-05 | `log`, `log-period`, `log-sheet`, `log-customize`, `log-taxonomy`, `voice-log` | sonnet + opus review | parallel group M8-C |
| T-M8-06 | `fertility`, `pregnancy`, `pregnancy-v2`, `pregnancy-tools`, `analysis-pregnancy`, `ivf`, `loss` | sonnet + opus review | parallel group M8-C |
| T-M8-07 | `postpartum`, `analysis-postpartum`, `baby-log`, `children`, `companions`, `companion-home`, `pelvic` | sonnet + opus review | parallel group M8-C |
| T-M8-08 | `record`, `record-export`, `health-record`, `labs`, `home-lab` | sonnet + opus review | parallel group M8-C |
| T-M8-09 | `vitals`, `checkups`, `care`, `reminders`, `conditions` | sonnet + opus review | parallel group M8-C |
| T-M8-10 | `menopause`, `teen`, `contraception` | sonnet + opus review | parallel group M8-C |
| T-M8-11 | `shop`, `web-shop`, `insurance`, `plus` | sonnet + opus review | parallel group M8-C |
| T-M8-12 | `services`, `city-services`, `city-join`, `doctors`, `web-directory`, `web-business` | sonnet + opus review | parallel group M8-C |
| T-M8-13 | `me`, `assistant`, `learn`, `articles`, `todo` | sonnet + opus review | parallel group M8-C |
| T-M8-14 | hardcoded Persian in `frontend/src` | sonnet + opus review | after the pilot |
| T-M8-15 | sweep + QA + stage | opus (haiku for re-runs) | last |
