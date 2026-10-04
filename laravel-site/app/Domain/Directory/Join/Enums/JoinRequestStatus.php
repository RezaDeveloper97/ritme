<?php

declare(strict_types=1);

namespace App\Domain\Directory\Join\Enums;

/**
 * Review state of a «ثبت مجموعه» request: every request starts pending; the admin review (L5-06) approves it (and
 * creates the place) or rejects it.
 */
enum JoinRequestStatus: string
{
    case Pending = 'pending';
    case Approved = 'approved';
    case Rejected = 'rejected';

    public function label(): string
    {
        return match ($this) {
            self::Pending => 'در انتظار بررسی',
            self::Approved => 'تأیید شده',
            self::Rejected => 'رد شده',
        };
    }

    public function color(): string
    {
        return match ($this) {
            self::Pending => 'warning',
            self::Approved => 'success',
            self::Rejected => 'gray',
        };
    }
}
