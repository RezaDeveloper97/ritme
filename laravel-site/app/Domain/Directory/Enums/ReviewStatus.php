<?php

declare(strict_types=1);

namespace App\Domain\Directory\Enums;

enum ReviewStatus: string
{
    case Pending = 'pending';
    case Approved = 'approved';
    case Rejected = 'rejected';

    public function label(): string
    {
        return match ($this) {
            self::Pending => 'در انتظار بررسی',
            self::Approved => 'تأییدشده',
            self::Rejected => 'ردشده',
        };
    }
}
