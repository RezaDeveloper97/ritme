<?php

declare(strict_types=1);

namespace App\Domain\Shop\Payment\Data;

use App\Domain\Shop\Payment\Enums\PaymentStatus;

/**
 * What happens after an order is stored: `redirectUrl` null = nothing to do online (cash on delivery); a future bank
 * gateway would return its payment page here. `status` is the payment status the order starts with.
 */
final readonly class PaymentStart
{
    public function __construct(public PaymentStatus $status, public ?string $redirectUrl = null) {}
}
