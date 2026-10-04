<?php

declare(strict_types=1);

namespace App\Domain\Directory\Enums;

enum PlaceStatus: string
{
    case Draft = 'draft';
    case Published = 'published';
    case Suspended = 'suspended';

    public function label(): string
    {
        return match ($this) {
            self::Draft => 'پیش‌نویس',
            self::Published => 'منتشرشده',
            self::Suspended => 'معلق',
        };
    }
}
