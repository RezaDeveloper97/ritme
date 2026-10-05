<?php

declare(strict_types=1);

namespace App\Domain\Shop\Payment\Enums;

/**
 * How an order is paid. Cash on delivery only for now (tasks/README.md decision); a bank gateway adds a case + gateway.
 */
enum PaymentMethod: string
{
    case CashOnDelivery = 'cash_on_delivery';

    public function label(): string
    {
        return match ($this) {
            self::CashOnDelivery => 'پرداخت در محل',
        };
    }
}
