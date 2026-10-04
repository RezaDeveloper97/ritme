<?php

declare(strict_types=1);

namespace App\Domain\Newsletter\Enums;

/**
 * Derived from the subscriber's timestamps (no status column): unsubscribed wins, then confirmed, else pending.
 */
enum SubscriptionStatus: string
{
    case Pending = 'pending';
    case Active = 'active';
    case Unsubscribed = 'unsubscribed';

    public function label(): string
    {
        return match ($this) {
            self::Pending => 'در انتظار تأیید',
            self::Active => 'فعال',
            self::Unsubscribed => 'لغو شده',
        };
    }
}
