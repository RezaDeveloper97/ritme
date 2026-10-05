# Shop / Cart (L6-04)

Session cart of the single-seller shop. The session holds only ids, quantities and the unit price seen when the line
was last checked (`Data\Cart` → `Contracts\CartRepository`, session implementation; ≤ 30 lines, ≤ 10 per line), so
it stays tiny and a client can never set a price.

- Every read goes through `Actions\ResolveCart`: products / variants and their stock are re-read **live** (uncached,
  `Queries\LiveCartCatalog`), vanished lines are dropped, quantities clamped to the stock left, sold-out lines kept but
  flagged and left out of the totals, price changes noted. Returns `Data\CartSummary` (DTO for the view).
- Mutations: `AddToCart` (variant resolved from size + colour with `variantFor`), `UpdateCartLine`, `RemoveFromCart`,
  `ClearCart`. Problems are `Exceptions\CartException` / notices with a `Enums\CartProblem` (copy in `lang/fa/shop.php`
  `cart.problems.*`).
- Shipping: `Data\ShippingRule` (flat fee and/or free over a threshold, both optional → «محاسبه در مرحله بعد»), bound
  in `ShopServiceProvider` from `config('shop.shipping.*')` until a shop settings group exists (L6-06).
- The header badge reads the plain cookie `ritme_cart_count` that `CartController` writes after every cart response.
