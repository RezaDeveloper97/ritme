<?php

declare(strict_types=1);

namespace App\Domain\Shop\Ordering\Actions;

use App\Domain\Shop\Catalog\Actions\AdjustStock;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Ordering\Enums\OrderStatus;
use App\Domain\Shop\Ordering\Models\Order;
use App\Domain\Shop\Payment\Enums\PaymentStatus;
use App\Support\Cache\NamespaceBumper;
use Illuminate\Database\ConnectionInterface;
use Illuminate\Database\Eloquent\Model;
use InvalidArgumentException;

/**
 * Moves an order along its lifecycle (admin, L6-06) — only the transitions of the state machine:
 *
 *   pending → confirmed | cancelled,  confirmed → shipped | cancelled,  shipped → delivered | cancelled (returned),
 *   delivered / cancelled → (final)
 *
 * One transaction; the status is switched with a conditional UPDATE (`WHERE status = from`), so two admins clicking
 * at the same time cannot both apply a change. Side effects:
 *  - delivered: cash on delivery is collected by the courier → `payment_status` becomes paid;
 *  - cancelled: stock is returned for the lines whose stock was decremented at checkout (`stock_tracked`; pre-/back-
 *    order lines were never taken from stock) through AdjustStock, and `sales_count` is lowered for every line (raised
 *    for every line by PlaceOrder). A line whose product / variant was deleted or deactivated meanwhile cannot be
 *    returned and is reported in the activity entry.
 * Every change is written to the activity log (`shop`, `shop.order.status`, no personal data in the properties).
 */
final class ChangeOrderStatus
{
    public const LOG = 'shop';

    private const FLOW = [
        'pending' => [OrderStatus::Confirmed, OrderStatus::Cancelled],
        'confirmed' => [OrderStatus::Shipped, OrderStatus::Cancelled],
        'shipped' => [OrderStatus::Delivered, OrderStatus::Cancelled],
        'delivered' => [],
        'cancelled' => [],
    ];

    public function __construct(
        private readonly ConnectionInterface $db,
        private readonly AdjustStock $stock,
        private readonly NamespaceBumper $bumper,
    ) {}

    /**
     * @return list<OrderStatus>
     */
    public static function next(OrderStatus $from): array
    {
        return self::FLOW[$from->value];
    }

    public static function allowed(OrderStatus $from, OrderStatus $to): bool
    {
        return in_array($to, self::next($from), true);
    }

    /**
     * @throws InvalidArgumentException when the state machine does not allow the change (or another admin changed
     *                                  the order first)
     */
    public function handle(Order $order, OrderStatus $status, ?Model $causer = null): Order
    {
        $from = $order->status;
        if (! self::allowed($from, $status)) {
            throw new InvalidArgumentException("Order status cannot change from {$from->value} to {$status->value}.");
        }

        /** @var array<string, mixed> $properties */
        $properties = $this->db->transaction(function () use ($order, $from, $status): array {
            $changes = ['status' => $status->value, 'updated_at' => now()];
            if ($status === OrderStatus::Delivered) {
                $changes['payment_status'] = PaymentStatus::Paid->value;
            }

            $updated = Order::query()->whereKey($order->id)->where('status', $from->value)->toBase()->update($changes);
            if ($updated !== 1) {
                throw new InvalidArgumentException('The order status was changed by someone else; reload the order.');
            }

            $properties = ['old' => ['status' => $from->value], 'attributes' => ['status' => $status->value]];
            if ($status === OrderStatus::Delivered) {
                $properties['attributes']['payment_status'] = PaymentStatus::Paid->value;
            }
            if ($status === OrderStatus::Cancelled) {
                $properties['stock'] = $this->restock($order);
            }

            return $properties;
        });

        if ($status === OrderStatus::Cancelled) {
            $this->bumper->bumpFor(new Product, ['shop']); // best-seller order follows sales_count
        }

        activity(self::LOG)
            ->causedBy($causer)
            ->performedOn($order)
            ->event('updated')
            ->withProperties($properties)
            ->log('shop.order.status');

        return $order->refresh();
    }

    /**
     * Returns the stock of the tracked lines and lowers `sales_count`.
     *
     * @return array{restored: list<int>, skipped: list<int>}
     */
    private function restock(Order $order): array
    {
        $restored = [];
        $skipped = [];
        foreach ($order->items()->get() as $item) {
            if ($item->product_id === null) {
                $skipped[] = $item->id;

                continue;
            }

            Product::query()->whereKey($item->product_id)->toBase()
                ->update(['sales_count' => $this->db->raw('CASE WHEN sales_count > '.$item->quantity.' THEN sales_count - '.$item->quantity.' ELSE 0 END')]);

            if (! $item->stock_tracked) {
                continue;
            }

            try {
                $returned = $this->stock->handle($item->product_id, $item->variant_id, $item->quantity);
            } catch (InvalidArgumentException) {
                $returned = false; // the variant was deleted: the product now has other variants
            }

            if ($returned) {
                $restored[] = $item->id;
            } else {
                $skipped[] = $item->id;
            }
        }

        return ['restored' => $restored, 'skipped' => $skipped];
    }
}
