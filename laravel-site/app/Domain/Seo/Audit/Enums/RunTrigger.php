<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit\Enums;

enum RunTrigger: string
{
    case Cli = 'cli';
    case Schedule = 'schedule';
    case Admin = 'admin';

    public function label(): string
    {
        return match ($this) {
            self::Cli => 'خط فرمان',
            self::Schedule => 'زمان‌بندی هفتگی',
            self::Admin => 'اجرای دستی از پنل',
        };
    }
}
