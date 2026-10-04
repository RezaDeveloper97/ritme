---
id: CB-TEEN-04b
title: Teen mode fertility copy leaks and profile rows
epic: TEEN
type: fullstack
status: todo
depends_on: [CB-TEEN-04]
parallel_group: TEEN-E
touches: [frontend/src/screens/calendar,frontend/src/screens/profile,frontend/src/screens/home,frontend/src/features/log-day,backend-go/internal/messages,backend-go/internal/cycle,backend-go/contract,frontend/messages,backend-go/resources/translations,backend-go/internal/i18n/testdata,docs/qa/canvas]
skills: [new-endpoint,new-fsd-slice]
boards: [nbl_Teen_Home.dc.html]
verify: cd backend-go && go vet ./... && go test ./... && golangci-lint run && make contract ROUTES=all && cd ../frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# CB-TEEN-04b — Teen mode fertility copy leaks and profile rows

## Why
CB-TEEN-04 QA found fertility copy reaching teen accounts outside the teen home, plus profile rows that don't fit teen mode (details in docs/qa/canvas/teen.md § CB-TEEN-04).

## Boards
- `nbl_Teen_Home.dc.html` (teen mode = no fertility copy anywhere)

## Scope
1. `/calendar` in teen mode: hide fertile-window / ovulation markers, the «پنجره باروری» legend and the fertile phase label (as the home does with `hideFertility`).
2. Backend: `/messages/daily` and `/cycle/today` return no fertility / ovulation copy for teen-mode accounts (extends bloom B-N2-11b's NoFertilityCopy); teen home stops calling `/messages/daily` (also removes the 400 flash for users without period data). Contract goldens updated.
3. `/profile` in teen mode: «همدم‌ها و خانواده» shows the parent link (not an inactive partner's name); hide «کارها و خریدها / سفارش‌ها»; log sheet hides «دمای پایه» (BBT) for teens.

## Out of scope
- Catalog copy rewrites in 00029_teen.sql (content + clinical review; proposals in the QA doc); auto-revoke/pause of pre-teen partner links (product decision).

## Acceptance
- A teen account sees no fertility/ovulation wording on home, calendar, daily message, cycle today or log sheet (int + unit tests); light + dark screenshots in docs/qa/canvas/teen.md.
- `verify` green.
