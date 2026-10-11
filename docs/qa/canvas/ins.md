## CB-INS-03

Worktree Go API :8341 + Next dev :3141 against `ins03_dev` (fresh goose 00082, user `09120000301`, 3 catalog insurers, policy with 5
coverages, 2 claims: one `docs_requested`, one `reviewing`). 390 px, fa, light + dark. Colours come from the app tokens, not the board hex.

| Board | Route | Light | Dark | Verdict | Fix / note |
| --- | --- | --- | --- | --- | --- |
| nbl_Ins_Home (`boards/nbl_Ins_Home.png`) | `/insurance` (policy exists) | `CB-INS-03/fa_insurance.home.light.png` | `CB-INS-03/fa_insurance.home.dark.png` | Matches: policy card, five actions, action card, remaining bars, recent claims, nav | Greeting «سلام سارا» omitted (no profile name read here); Rec_Home link not added |
| nbl_Ins_Home (empty state) | `/insurance` (no policy) | `CB-INS-03/fa_insurance.setup.light.png` | `CB-INS-03/fa_insurance.setup.dark.png` | New (no board): setup form + link to the info page; no bottom nav | — |
| nbl_Ins_Coverage (`boards/nbl_Ins_Coverage.png`) | `/insurance/coverage` | `CB-INS-03/fa_insurance_coverage.coverage.light.png` | `CB-INS-03/fa_insurance_coverage.coverage.dark.png` | Matches: deductible tiles, caps + usage, waiting notes, not-covered note | «دانلود متن کامل بیمه‌نامه» omitted: the API stores no policy document |
