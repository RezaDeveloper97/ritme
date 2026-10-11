## CB-INS-03

Worktree Go API :8341 + Next dev :3141 against `ins03_dev` (fresh goose 00082, user `09120000301`, 3 catalog insurers, policy with 5
coverages, 2 claims: one `docs_requested`, one `reviewing`). 390 px, fa, light + dark. Colours come from the app tokens, not the board hex.

| Board | Route | Light | Dark | Verdict | Fix / note |
| --- | --- | --- | --- | --- | --- |
| nbl_Ins_Home (`boards/nbl_Ins_Home.png`) | `/insurance` (policy exists) | `CB-INS-03/fa_insurance.home.light.png` | `CB-INS-03/fa_insurance.home.dark.png` | Matches: policy card, five actions, action card, remaining bars, recent claims, nav | Greeting «سلام سارا» omitted (no profile name read here); Rec_Home link not added |
| nbl_Ins_Home (empty state) | `/insurance` (no policy) | `CB-INS-03/fa_insurance.setup.light.png` | `CB-INS-03/fa_insurance.setup.dark.png` | New (no board): setup form + link to the info page; no bottom nav | — |
| nbl_Ins_Coverage (`boards/nbl_Ins_Coverage.png`) | `/insurance/coverage` | `CB-INS-03/fa_insurance_coverage.coverage.light.png` | `CB-INS-03/fa_insurance_coverage.coverage.dark.png` | Matches: deductible tiles, caps + usage, waiting notes, not-covered note | «دانلود متن کامل بیمه‌نامه» omitted: the API stores no policy document |

## CB-INS-06

Next dev :3146 with the API mocked at the Playwright network layer (4 centres: 2 direct, 2 reimbursed, one without coordinates). 390 px, fa, light + dark. No `NEXT_PUBLIC_NESHAN_KEY` in dev, so the map card is not drawn (list-only fallback, DECISIONS #16); the light shot also shows the PWA install banner.

| Board | Route | Light | Dark | Verdict | Fix / note |
| --- | --- | --- | --- | --- | --- |
| nbl_Ins_Centers (`boards/nbl_Ins_Centers.png`) | `/insurance/centers` | `CB-INS-06/fa_insurance_centers.centers.light.png` | `CB-INS-06/fa_insurance_centers.centers.dark.png` | Matches: back header, search, type chips, cards with direct/reimbursed badge + call/locate buttons, deductible note; map card appears only with a Neshan key | Distance (km) omitted: the API returns none and we never read her location; «نقشه» toggle omitted |
## CB-INS-05

Next dev :3151 with the `/api/v1/insurance/*` calls mocked at the Playwright network layer (no private DB copy; renewal questionnaire, 6 of 9 answers from record, one doc-requested claim). 390 px, fa, light + dark. Colours come from the app tokens, not the board hex.

| Board | Route | Light | Dark | Verdict | Fix / note |
| --- | --- | --- | --- | --- | --- |
| nbl_Ins_Health (`boards/nbl_Ins_Health.png`) | `/insurance/questionnaire` (section 2) | `CB-INS-05/fa_questionnaire.history.light.png` | `CB-INS-05/fa_questionnaire.history.dark.png` | Matches: section stepper + «بخش ۲ از ۴», yes/no rows with «از پرونده سلامت» tag, explanation card with record items, warning, continue | Board warning «پاسخ‌ها فقط برای بیمه‌گر ارسال می‌شود» replaced: nothing is sent; answers stay in her record and she can export a PDF |
| nbl_Ins_Health (review) | `/insurance/questionnaire` (last step) | `CB-INS-05/fa_questionnaire.review.light.png` | `CB-INS-05/fa_questionnaire.review.dark.png` | New (no board): per-section summary, edit links, confirm | Confirm disabled while a question is unanswered |
| nbl_Ins_Health (confirmed) | `/insurance/questionnaire` | `CB-INS-05/fa_questionnaire.done.light.png` | `CB-INS-05/fa_questionnaire.done.dark.png` | New: confirmation, on-device PDF export, link to status | PDF drawn on the device (shared/lib/pdf), never uploaded |
| nbl_Ins_Status (`boards/nbl_Ins_Status.png`) | `/insurance/status` | `CB-INS-05/fa_status.light.png` | `CB-INS-05/fa_status.dark.png` | Matches: questionnaire card with 4-row timeline, open claims | Steps are recorded by her (sheet with step chips + date); no insurer ETA («۲ روز کاری») since nothing is fetched |
