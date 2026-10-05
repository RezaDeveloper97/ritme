<?php

declare(strict_types=1);

namespace App\Domain\Shop\Payment\Contracts;

use App\Domain\Shop\Payment\Data\PaymentStart;
use App\Domain\Shop\Payment\Enums\PaymentMethod;
use App\Support\Money\Money;

/**
 * A way to pay for an order. Checkout asks `accepts()` before the order is stored (inside the order transaction) and
 * calls `start()` after it is committed. Only CashOnDeliveryGateway exists (COD-only decision).
 */
interface PaymentGateway
{
    public function method(): PaymentMethod;

    /** Highest total this gateway takes, null = no limit. */
    public function limit(): ?Money;

    public function accepts(Money $total): bool;

    public function start(string $orderCode, Money $total): PaymentStart;
}
