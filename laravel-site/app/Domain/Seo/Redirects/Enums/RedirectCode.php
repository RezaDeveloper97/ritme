<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Enums;

/**
 * Status codes a redirect row may answer with. 410 has no target: the old URL is gone for good.
 */
enum RedirectCode: int
{
    case Permanent = 301;
    case Found = 302;
    case Temporary = 307;
    case Gone = 410;

    public function label(): string
    {
        return match ($this) {
            self::Permanent => '۳۰۱ — دائمی',
            self::Found => '۳۰۲ — موقت',
            self::Temporary => '۳۰۷ — موقت (حفظ متد)',
            self::Gone => '۴۱۰ — حذف‌شده',
        };
    }

    public function isGone(): bool
    {
        return $this === self::Gone;
    }

    /**
     * @return array<int, string>
     */
    public static function options(): array
    {
        $options = [];
        foreach (self::cases() as $case) {
            $options[$case->value] = $case->label();
        }

        return $options;
    }
}
