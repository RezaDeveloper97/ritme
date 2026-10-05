# Shop / Ordering (L6-05)

Checkout and orders of the single-seller shop. Cash on delivery only (`Shop\Payment`, `PaymentGateway` contract).

- `Actions\PlaceOrder` — one DB transaction: re-resolves the session cart LIVE (`Cart\Actions\ResolveCart`), refuses
  (`Exceptions\CheckoutException` + `Enums\CheckoutProblem`, copy in `lang/fa/shop.php` `checkout.problems.*`) an
  empty / changed (`Support\CartSignature`) / sold-out cart or a total above the COD cap, decrements stock line by line
  with `Catalog\Actions\AdjustStock` (atomic, no oversell), writes `Models\Order` + `OrderItem` snapshots, raises
  `sales_count`. Idempotent: the checkout form token's hash is the unique `idempotency_key`. After commit: `ClearCart`,
  `Events\OrderPlaced` → `Listeners\SendOrderNotifications` (queued team mail + customer SMS via `SmsSender`).
- `Support\OrderCode` — unguessable public code (`XXXX-XXXX-XXXX`) of `/shop/order/{code}`; `Support\CheckoutSession`
  — the one-time form token + the codes this session placed (only their owner sees the recipient rows).
- `Support\DeliverySlots` — preferred day + window (a preference the shop confirms by phone), `Support\IranProvinces`
  — local province/city lists (no external service).
- `Queries\FindOrder` — uncached lookups; `Data\OrderData` is already masked for the public page.
- Minimal personal data: name, mobile, province/city, address, postal code, note. No email, IP or user agent.
