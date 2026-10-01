---
id: B-N2-11b
title: N2 stage smoke follow-ups (checkout code label, teen/menopause copy, version)
milestone: N2
type: fullstack
status: todo
depends_on: [B-N2-11]
parallel_group: N2-L2
touches: [frontend/src/screens/plus-checkout,frontend/src/screens/home,frontend/src/screens/profile,frontend/src/widgets/checkups-card,backend-go/internal/messages,backend-go/internal/checkups]
skills: [verify-all]
verify: cd backend-go && go vet ./... && go test ./... && make lint && cd .. && cd frontend && npm run typecheck && npm run lint && npm run fsd:lint && npm run lint:styles && npm run lint:dark && npm run test && npm run build
---

# B-N2-11b — N2 stage smoke follow-ups (checkout code label, teen/menopause copy, version)

## Why
Low bugs from the N2 stage smoke (`docs/qa/bloom/n2-stage.md`).

## Scope
- B-2: checkout shows «کد تخفیف اعمال شد» when the trial offer won and the code was not used (`discount_source=trial_offer`, `discount_code=null`) — say the offer was applied instead (`screens/plus-checkout/ui/CheckoutPage.tsx`).
- B-3: teen home phase card talks about «پنجره باروری» — message engine must not emit fertility copy for teen (`internal/messages`), or the card filters it.
- B-4: menopause home checkup card times by cycle day («روز ۷ تا ۱۰ سیکل») — menopause users get non-cycle timing (`internal/checkups` / `widgets/checkups-card`).
- B-5: Me hub footer hard-codes «نسخه ۱.۰.۰» — read the stamped app version (PWA version stamped at prebuild).
- (B-1 male landing on the women cycle home is B-N4-05 scope — not here.)

## Out of scope
- 

## Acceptance
- Each bug fixed with a test where logic changed; light + dark screenshot of the touched screens in `docs/qa/bloom/B-N2-11b/`
- `verify` green
