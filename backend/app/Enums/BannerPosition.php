<?php

namespace App\Enums;

/**
 * Fixed slots on the app home page where a banner slideshow can appear. Each
 * slot stays empty (renders nothing) until an admin publishes a banner for it,
 * so positions are safe to reference from the frontend even when unused.
 */
enum BannerPosition: string
{
    case HOME_TOP = 'home_top';
    case HOME_MIDDLE = 'home_middle';
    case HOME_BOTTOM = 'home_bottom';

    public function label(string $locale = 'en'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::HOME_TOP => 'خانه - بالا',
                self::HOME_MIDDLE => 'خانه - میانه',
                self::HOME_BOTTOM => 'خانه - پایین',
            },
            default => match ($this) {
                self::HOME_TOP => 'Home - Top',
                self::HOME_MIDDLE => 'Home - Middle',
                self::HOME_BOTTOM => 'Home - Bottom',
            },
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
