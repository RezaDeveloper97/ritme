---
id: L6-05
title: Checkout (cash on delivery), orders, done page
milestone: L6
type: fullstack
status: todo
depends_on: [L6-04]
parallel_group: L6-C
touches: [app/Domain/Shop/Ordering,app/Domain/Shop/Payment,database/migrations,app/Http/Controllers/Shop/CheckoutController.php,app/Http/Requests/CheckoutRequest.php,resources/views/pages/shop/checkout.blade.php,resources/views/pages/shop/done.blade.php,app/Notifications,tests/Feature/Shop/CheckoutTest.php]
skills: []
verify: composer verify && node tools/shot.mjs --design shop-checkout.html --route /shop/checkout && node tools/shot.mjs --design shop-done.html --route /shop/order/DEMO
---

# L6-05 — Checkout (cash on delivery), orders, done page

## Scope
- `/shop/checkout` (noindex): contact + address (province/city lists local data), delivery method, notes; payment
  `cash_on_delivery` through `PaymentGateway` contract (`CashOnDeliveryGateway` only — YAGNI).
- `PlaceOrder` action: DB transaction, re-price, lock + decrement stock, `Order` (code, status pending|confirmed|
  shipped|delivered|cancelled, totals as Money), `OrderItem` snapshots, `OrderPlaced` event → queued notifications
  (admin mail, customer SMS via `SmsSender` log driver).
- `/shop/order/{code}` done page (noindex, unguessable code + session check) matching `shop-done.html`.
- Idempotency: double-submit protection (token), rate limit.

## Acceptance
- Concurrent stock test (two orders, one unit) → one succeeds; diffs < 3%; tests green.
