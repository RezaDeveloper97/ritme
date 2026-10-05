<?php

declare(strict_types=1);

namespace App\Domain\Shop\Ordering\Actions;

use App\Domain\Shop\Cart\Actions\ClearCart;
use App\Domain\Shop\Cart\Actions\ResolveCart;
use App\Domain\Shop\Cart\Contracts\CartCatalog;
use App\Domain\Shop\Cart\Data\CartItemData;
use App\Domain\Shop\Cart\Data\CartSummary;
use App\Domain\Shop\Catalog\Actions\AdjustStock;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Ordering\Data\CheckoutSubmission;
use App\Domain\Shop\Ordering\Data\OrderPlacement;
use App\Domain\Shop\Ordering\Enums\CheckoutProblem;
use App\Domain\Shop\Ordering\Events\OrderPlaced;
use App\Domain\Shop\Ordering\Exceptions\CheckoutException;
use App\Domain\Shop\Ordering\Models\Order;
use App\Domain\Shop\Ordering\Support\CartSignature;
use App\Domain\Shop\Ordering\Support\CheckoutSession;
use App\Domain\Shop\Ordering\Support\OrderCode;
use App\Domain\Shop\Payment\Contracts\PaymentGateway;
use App\Domain\Shop\Payment\Data\PaymentStart;
use Illuminate\Contracts\Events\Dispatcher;
use Illuminate\Database\ConnectionInterface;
use Illuminate\Database\UniqueConstraintViolationException;
use RuntimeException;

/**
 * Places a cash-on-delivery order from the session cart, all-or-nothing in ONE database transaction:
 *
 *  1. the cart is re-resolved against the LIVE catalog (ResolveCart: prices + stock re-read, never from the client);
 *  2. refused (CheckoutException, Persian copy) when it is empty, differs from what the checkout page showed
 *     (CartSignature: price / quantity / shipping changed), holds sold-out lines, or the total is above the COD cap
 *     (PaymentGateway::accepts);
 *  3. every line's stock is decremented with AdjustStock (one conditional UPDATE each — two concurrent orders for the
 *     last unit cannot both succeed); one refusal rolls every earlier decrement back. Pre-/back-order products are
 *     decremented while stock lasts and sold beyond it (`stock_tracked` false);
 *  4. the Order + OrderItem snapshots are written and `sales_count` is raised.
 *
 * Idempotent per checkout form: the token's hash is the unique `idempotency_key`, so a double submit (sequential or a
 * concurrent race that hits the unique index) returns the first order — `duplicate` true, nothing placed twice.
 * After commit: the cart is cleared, OrderPlaced is dispatched (queued notifications) and the gateway is started.
 */
final class PlaceOrder
{
    private const CODE_ATTEMPTS = 5;

    public function __construct(
        private readonly ConnectionInterface $db,
        private readonly ResolveCart $resolve,
        private readonly CartCatalog $catalog,
        private readonly AdjustStock $stock,
        private readonly ClearCart $clear,
        private readonly PaymentGateway $gateway,
        private readonly Dispatcher $events,
    ) {}

    public function handle(CheckoutSubmission $submission): OrderPlacement
    {
        $key = CheckoutSession::key($submission->token);
        $existing = $this->existing($key);
        if ($existing !== null) {
            return $existing;
        }

        try {
            /** @var Order $order */
            $order = $this->db->transaction(fn (): Order => $this->place($submission, $key));
        } catch (UniqueConstraintViolationException $e) {
            return $this->existing($key) ?? throw $e;
        }

        $this->clear->handle();
        $this->events->dispatch(new OrderPlaced($order));

        return new OrderPlacement($order, $this->gateway->start($order->code, $order->total));
    }

    private function place(CheckoutSubmission $submission, string $key): Order
    {
        $cart = $this->resolve->handle();
        $this->assertPlaceable($cart, $submission->signature);

        $total = $cart->total;
        if (! $this->gateway->accepts($total)) {
            throw new CheckoutException(CheckoutProblem::CodLimit, ['amount' => $this->gateway->limit()?->format() ?? '']);
        }

        $buyable = array_values(array_filter($cart->items, static fn (CartItemData $i): bool => $i->available));
        $products = $this->catalog->find(array_values(array_unique(array_map(static fn (CartItemData $i): int => $i->productId, $buyable))));

        $lines = [];
        foreach ($buyable as $item) {
            $ignoresStock = ($products[$item->productId] ?? null)?->ignoresStock() ?? false;
            $tracked = $this->stock->handle($item->productId, $item->variantId, -$item->quantity);
            if (! $tracked && ! $ignoresStock) {
                throw new CheckoutException(CheckoutProblem::OutOfStock, ['name' => $item->title]);
            }
            Product::query()->whereKey($item->productId)->toBase()->increment('sales_count', $item->quantity);
            $lines[] = [$item, $tracked];
        }

        $order = Order::query()->create([
            'code' => $this->uniqueCode(),
            'idempotency_key' => $key,
            'payment_method' => $this->gateway->method(),
            'subtotal' => $cart->subtotal,
            'shipping_fee' => $cart->shipping->fee,
            'total' => $total,
            'items_count' => $cart->count,
            'recipient_name' => $submission->recipientName,
            'mobile' => $submission->mobile,
            'province' => $submission->province,
            'city' => $submission->city,
            'address' => $submission->address,
            'postal_code' => $submission->postalCode,
            'note' => $submission->note,
            'delivery_date' => $submission->deliveryDate->format('Y-m-d'),
            'delivery_window' => $submission->deliveryWindow,
            'discreet_packaging' => $submission->discreetPackaging,
            'is_demo' => $cart->hasDemo(),
        ]);

        foreach ($lines as [$item, $tracked]) {
            $order->items()->create([
                'product_id' => $item->productId,
                'variant_id' => $item->variantId,
                'title' => mb_substr($item->title, 0, 191),
                'variant_label' => $item->variantLabel === null ? null : mb_substr($item->variantLabel, 0, 120),
                'unit_price' => $item->unitPrice,
                'quantity' => $item->quantity,
                'line_total' => $item->lineTotal(),
                'cover_media_id' => $item->coverMediaId,
                'illustration' => $item->illustration,
                'stock_tracked' => $tracked,
            ]);
        }

        return $order;
    }

    private function assertPlaceable(CartSummary $cart, string $signature): void
    {
        $soldOut = array_values(array_filter($cart->items, static fn (CartItemData $i): bool => ! $i->available));
        if ($soldOut !== []) {
            throw new CheckoutException(CheckoutProblem::Unavailable, ['name' => $soldOut[0]->title]);
        }

        if (! $cart->canCheckout()) {
            throw new CheckoutException(CheckoutProblem::EmptyCart);
        }

        if ($cart->notices !== [] || ! hash_equals(CartSignature::of($cart), $signature)) {
            $notices = $cart->notices;
            foreach ($cart->items as $item) {
                if ($item->notice !== null) {
                    $notices[] = $item->notice;
                }
            }

            throw new CheckoutException(CheckoutProblem::CartChanged, notices: $notices);
        }
    }

    private function existing(string $key): ?OrderPlacement
    {
        $order = Order::query()->where('idempotency_key', $key)->first();

        return $order === null ? null : new OrderPlacement($order, new PaymentStart($order->payment_status), duplicate: true);
    }

    private function uniqueCode(): string
    {
        for ($i = 0; $i < self::CODE_ATTEMPTS; $i++) {
            $code = OrderCode::generate();
            if (! Order::query()->where('code', $code)->exists()) {
                return $code;
            }
        }

        throw new RuntimeException('Could not generate a unique order code.');
    }
}
