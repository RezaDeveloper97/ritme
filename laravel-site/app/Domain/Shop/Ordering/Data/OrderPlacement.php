<?php

declare(strict_types=1);

namespace App\Domain\Shop\Ordering\Data;

use App\Domain\Shop\Ordering\Models\Order;
use App\Domain\Shop\Payment\Data\PaymentStart;

/**
 * Result of PlaceOrder: the order, what the payment gateway wants next (COD: nothing) and whether this was a repeated
 * submit of an order that already exists (same idempotency token → same order, nothing placed twice).
 */
final readonly class OrderPlacement
{
    public function __construct(public Order $order, public PaymentStart $payment, public bool $duplicate = false) {}
}
