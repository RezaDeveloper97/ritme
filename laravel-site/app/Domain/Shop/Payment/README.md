# Shop / Payment (L6-05)

How an order is paid. Decision (tasks/README.md): **cash on delivery only**, behind the `Contracts\PaymentGateway`
contract so a bank gateway can be added later without touching checkout.

- `Gateways\CashOnDeliveryGateway` — the only gateway (bound in `ShopServiceProvider`). Accepts a total up to the COD
  cap (`ShopSettings.cod_max_amount` from admin → تنظیمات فروشگاه, tomans, null = no cap; `config('shop.cod_max_amount')`
  as fallback). `start()` needs no redirect: the order stays `unpaid` until the courier is paid.
- Never claim "paid" on the site: `Enums\PaymentStatus::Unpaid` reads «پرداخت هنگام تحویل».
