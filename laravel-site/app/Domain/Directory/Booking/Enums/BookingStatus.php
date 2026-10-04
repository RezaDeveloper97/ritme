<?php

declare(strict_types=1);

namespace App\Domain\Directory\Booking\Enums;

/**
 * Life of a booking request: every request starts `new`; the team / place confirms the time by phone (`confirmed`),
 * or it is `cancelled`; after the visit it is `done` (admin: L5-06).
 */
enum BookingStatus: string
{
    case New = 'new';
    case Confirmed = 'confirmed';
    case Cancelled = 'cancelled';
    case Done = 'done';

    public function label(): string
    {
        return match ($this) {
            self::New => 'در انتظار تأیید مجموعه',
            self::Confirmed => 'تأیید شده',
            self::Cancelled => 'لغو شده',
            self::Done => 'انجام شده',
        };
    }

    public function color(): string
    {
        return match ($this) {
            self::New => 'warning',
            self::Confirmed => 'success',
            self::Cancelled => 'gray',
            self::Done => 'info',
        };
    }
}
