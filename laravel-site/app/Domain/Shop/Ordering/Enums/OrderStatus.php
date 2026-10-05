<?php

declare(strict_types=1);

namespace App\Domain\Shop\Ordering\Enums;

/**
 * Lifecycle of an order. Placed orders start `pending` (the shop calls to confirm); the admin moves them on (L6-06).
 */
enum OrderStatus: string
{
    case Pending = 'pending';
    case Confirmed = 'confirmed';
    case Shipped = 'shipped';
    case Delivered = 'delivered';
    case Cancelled = 'cancelled';

    public function label(): string
    {
        return match ($this) {
            self::Pending => 'در انتظار تأیید',
            self::Confirmed => 'تأیید شد',
            self::Shipped => 'ارسال شد',
            self::Delivered => 'تحویل شد',
            self::Cancelled => 'لغو شد',
        };
    }

    /** Position on the done page's timeline (cancelled has none). */
    public function step(): int
    {
        return match ($this) {
            self::Pending => 1,
            self::Confirmed => 2,
            self::Shipped => 3,
            self::Delivered => 4,
            self::Cancelled => 0,
        };
    }
}
