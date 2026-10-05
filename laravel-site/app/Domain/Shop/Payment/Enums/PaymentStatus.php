<?php

declare(strict_types=1);

namespace App\Domain\Shop\Payment\Enums;

/**
 * Whether the money has been received. Cash on delivery stays `unpaid` until the courier hands the cash over (set in
 * the admin, L6-06) — the site never says "paid" before that.
 */
enum PaymentStatus: string
{
    case Unpaid = 'unpaid';
    case Paid = 'paid';

    public function label(): string
    {
        return match ($this) {
            self::Unpaid => 'پرداخت هنگام تحویل',
            self::Paid => 'پرداخت شد',
        };
    }
}
