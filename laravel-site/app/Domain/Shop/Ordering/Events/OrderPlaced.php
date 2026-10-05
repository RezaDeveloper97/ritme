<?php

declare(strict_types=1);

namespace App\Domain\Shop\Ordering\Events;

use App\Domain\Shop\Ordering\Models\Order;
use Illuminate\Contracts\Events\ShouldDispatchAfterCommit;
use Illuminate\Foundation\Events\Dispatchable;

/**
 * An order was stored (PlaceOrder). Dispatched after the order transaction commits; SendOrderNotifications queues the
 * team mail and the customer SMS.
 */
final class OrderPlaced implements ShouldDispatchAfterCommit
{
    use Dispatchable;

    public function __construct(public readonly Order $order) {}
}
