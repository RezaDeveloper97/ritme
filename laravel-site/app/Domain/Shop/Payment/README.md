# Shop / Payment (L6-05)

How an order is paid. Decision (tasks/README.md): **cash on delivery only**, behind the `Contracts\PaymentGateway`
contract so a bank gateway can be added later without touching checkout.

- `Gateways\CashOnDeliveryGateway` — the only gateway (bound in `ShopServiceProvider`). Accepts a total up to the COD
  cap (`config('shop.cod_max_amount')`, tomans, null = no cap — `ShopSettings.cod_max_amount` once a shop settings
  group exists, L6-06). `start()` needs no redirect: the order stays `unpaid` until the courier is paid.
- Never claim "paid" on the site: `Enums\PaymentStatus::Unpaid` reads «پرداخت هنگام تحویل».
