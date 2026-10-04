<?php

declare(strict_types=1);

namespace App\Domain\Blog\Enums;

enum PostStatus: string
{
    case Draft = 'draft';
    case Scheduled = 'scheduled';
    case Published = 'published';
    case Archived = 'archived';

    public function label(): string
    {
        return match ($this) {
            self::Draft => 'پیش‌نویس',
            self::Scheduled => 'زمان‌بندی‌شده',
            self::Published => 'منتشرشده',
            self::Archived => 'بایگانی',
        };
    }
}
